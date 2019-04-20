package business

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"gitlab.wallstcn.com/baoer/flash/flashcommon"
	"gitlab.wallstcn.com/baoer/flash/flashcommon/messages"
	"gitlab.wallstcn.com/baoer/flash/flashstd/json"
	"gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
)

func PubLockLimitUpEvt(stockData *types.StockData, limitUpPrice float64) {
	niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
	niceStockEvt.PriceAction = flashcommon.EnumPriceActionLockLimitUp
	uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)

	notifyPoolService(uglyStockEvt)

	if isNewStock(stockData.Sym) { // 新股开板回封，当天只提醒一次
		if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
			redislogger.Errorf("Failed to get events for stock [%s] from redis, err: [%v]", stockData.Sym, err)
		} else {
			notified := false
			for _, evt := range evts {
				if evt.PriceAction == flashcommon.EnumPriceActionFreshStockLockLimitUpAgain {
					notified = true
					break
				}
			}
			if !notified {
				niceStockEvt.PriceAction = flashcommon.EnumPriceActionFreshStockLockLimitUpAgain
				uglyStockEvt = NiceStockEvtToUglyStockEvt(niceStockEvt)
				notifyEvtService(uglyStockEvt, stockData)
				cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
				if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
					redislogger.Errorf("PubLockLimitUpEvt, failed to save event history, err: %v", err)
				}
			}
		}
	} else {
		notifyEvtService(uglyStockEvt, stockData)
	}

	redislogger.Printf("=> Lock Limit Up, limitUpPrice: %.2f, stockData: %s", limitUpPrice, stockData.ToString())
}

func PubComeOffLimitUpEvt(stockData *types.StockData, limitUpPrice float64) {
	if isNewStock(stockData.Sym) { // 新股开板，每次都提醒 PoolService，但是只发一次给 EventService
		niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
		niceStockEvt.PriceAction = flashcommon.EnumPriceActionFreshStockComeOffLimitUp
		uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
		notifyPoolService(uglyStockEvt)

		if err := g.FreshStockCache.MarkForDeletion(stockData.Sym); err != nil {
			redislogger.ErrorfToSlot(1, "Failed to mark stock: %s for deletion, err: %v", stockData.Sym, err)
		}
		if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
			redislogger.Errorf("Failed to get events for stock [%s] from redis, err: [%v]", stockData.Sym, err)
		} else {
			notified := false
			for _, evt := range evts {
				if evt.PriceAction == flashcommon.EnumPriceActionFreshStockComeOffLimitUp {
					notified = true
					break
				}
			}
			if !notified {
				notifyEvtService(uglyStockEvt, stockData)
				cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
				if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
					redislogger.Errorf("PubComeOffLimitUpEvt, failed to save event history, err: %v", err)
				}
			}
		}
	} else {
		niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
		niceStockEvt.PriceAction = flashcommon.EnumPriceActionComeOffLimitUp
		uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)

		notifyPoolService(uglyStockEvt)
		notifyEvtService(uglyStockEvt, stockData)
	}

	redislogger.Printf("=> Come Off Limit Up, limitUpPrice: %.2f, stockData: %s", limitUpPrice, stockData.ToString())
}

func PubAboutToComeOffLimitUpEvt(stockData *types.StockData, accumVol int64) {
	if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
		redislogger.Errorf("Failed to get events for stock [%s] from redis, err: [%v]", stockData.Sym, err)
	} else {
		publishedInLastThreeMinutes := false
		for _, evt := range evts {
			if evt.PriceAction == flashcommon.EnumPriceActionNearLimitUpBreach && time.Now().Unix()-evt.PublishedAt < 3*60 {
				publishedInLastThreeMinutes = true
				break
			}
		}
		if !publishedInLastThreeMinutes {
			niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
			niceStockEvt.PriceAction = flashcommon.EnumPriceActionNearLimitUpBreach
			uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
			notifyEvtService(uglyStockEvt, stockData)

			// save history
			cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
			if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
				redislogger.Errorf("PubAboutToComeOffLimitUpEvt, failed to save event history, err: %v", err)
			}

			redislogger.Printf("=> About to Come Off Limit Up, accumVol: %d stockData: %s", accumVol, stockData.ToString())
		}
	}
}

func PubApproachingLimitUpEvt(stockData *types.StockData, stockStat *types.StockStat, style string,
	upThresholdPrice, limitUpPrice float64) {
	niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
	niceStockEvt.PriceAction = flashcommon.EnumPriceActionApproachLimitUp
	uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
	notifyEvtService(uglyStockEvt, stockData)

	// save hisotry
	cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
	if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
		redislogger.Errorf("PubApproachingLimitUpEvt, failed to save event history, err: %v", err)
	}

	redislogger.Printf("=> Approaching Limit Up [%s], stockData: %s", style, stockData.ToString())
}

func PubSurgeEvent(stockData *types.StockData) {
	niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
	niceStockEvt.PriceAction = flashcommon.EnumPriceActionSurge
	uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
	notifyPoolService(uglyStockEvt)

	if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
		redislogger.Errorf("Failed to get events for stock[%s] from redis, err: [%v]", stockData.Sym, err)
	} else {
		publishedInLastThreeMinutes := false
		for _, evt := range evts {
			if evt.PriceAction == flashcommon.EnumPriceActionSurge && time.Now().Unix()-evt.PublishedAt < 3*60 {
				publishedInLastThreeMinutes = true
				break
			}
		}
		if !publishedInLastThreeMinutes {
			niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
			niceStockEvt.PriceAction = flashcommon.EnumPriceActionSurge
			uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
			notifyEvtService(uglyStockEvt, stockData)

			// save hisotry
			cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
			if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
				redislogger.Errorf("PubSurgeEvent, failed to save event history, err: %v", err)
			}

			redislogger.Printf("=> Surge, stockData: %s", stockData.ToString())
		}
	}
}

func PubPlummetEvt(stockData *types.StockData) {
	if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
		redislogger.Errorf("Failed to get events for stock[%s] from redis, err: [%v]", stockData.Sym, err)
	} else {
		publishedInLastThreeMinutes := false
		for _, evt := range evts {
			if evt.PriceAction == flashcommon.EnumPriceActionPlummet && time.Now().Unix()-evt.PublishedAt < 3*60 {
				publishedInLastThreeMinutes = true
				break
			}
		}
		if !publishedInLastThreeMinutes {
			niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
			niceStockEvt.PriceAction = flashcommon.EnumPriceActionPlummet
			uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
			notifyEvtService(uglyStockEvt, stockData)

			// save hisotry
			cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
			if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
				redislogger.Errorf("PubPlummetEvt, failed to save event history, err: %v", err)
			}
			redislogger.Printf("=> Plummet, stockData: %s", stockData.ToString())
		}
	}
}

func PubApproachingLimitDownEvt(stockData *types.StockData, stockStat *types.StockStat, style string) {
	if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
		redislogger.Errorf("Failed to get events for stock[%s] from redis, err: [%v]", stockData.Sym, err)
	} else {
		publishedInLastThreeMinutes := false
		for _, evt := range evts {
			if evt.PriceAction == flashcommon.EnumPriceActionApproachLimitDown && time.Now().Unix()-evt.PublishedAt < 3*60 {
				publishedInLastThreeMinutes = true
				break
			}
		}
		if !publishedInLastThreeMinutes {
			niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
			niceStockEvt.PriceAction = flashcommon.EnumPriceActionApproachLimitDown
			uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
			notifyEvtService(uglyStockEvt, stockData)

			// save hisotry
			cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
			if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
				redislogger.Errorf("PubApproachingLimitDownEvt, failed to save event history, err: %v", err)
			}
			redislogger.Printf("=> Approaching Limit Down [%s], stockData: %s", style, stockData.ToString())
		}
	}
}

func PubLockLimitDownEvt(stockData *types.StockData, limitDownPrice float64) {
	niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
	niceStockEvt.PriceAction = flashcommon.EnumPriceActionLockLimitDown
	uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)

	notifyPoolService(uglyStockEvt)
	notifyEvtService(uglyStockEvt, stockData)

	redislogger.Printf("=> Lock Limit Down, stockData: %s", stockData.ToString())
}

func PubAboutToComeOffLimitDownEvt(stockData *types.StockData, accumVol int64) {
	if evts, err := g.StockEvtCache.GetEvts(stockData.Sym); err != nil {
		redislogger.Errorf("Failed to get events for stock[%s] from redis, err: [%v]", stockData.Sym, err)
	} else {
		publishedInLastThreeMinutes := false
		for _, evt := range evts {
			if evt.PriceAction == flashcommon.EnumPriceActionNearLimitDownBreach && time.Now().Unix()-evt.PublishedAt < 3*60 {
				publishedInLastThreeMinutes = true
				break
			}
		}
		if !publishedInLastThreeMinutes {
			niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
			niceStockEvt.PriceAction = flashcommon.EnumPriceActionNearLimitDownBreach
			uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)
			notifyEvtService(uglyStockEvt, stockData)

			// save hisotry
			cacheStockEvt := NiceStockEvtToCacheStockEvt(niceStockEvt)
			if err := g.StockEvtCache.PutEvt(stockData.Sym, cacheStockEvt); err != nil {
				redislogger.Errorf("PubAboutToComeOffLimitDownEvt, failed to save event history, err: %v", err)
			}

			redislogger.Printf("=> About to Come Off Limit Down, stockData: %s", stockData.ToString())
		}
	}
}

func PubComeOffLimitDownEvt(stockData *types.StockData, limitDownPrice float64) {
	niceStockEvt := CreateNiceStockEvtFromStockData(stockData)
	niceStockEvt.PriceAction = flashcommon.EnumPriceActionComeOffLimitDown
	uglyStockEvt := NiceStockEvtToUglyStockEvt(niceStockEvt)

	notifyPoolService(uglyStockEvt)
	notifyEvtService(uglyStockEvt, stockData)

	redislogger.Printf("=> Come Off Limit Down, stockData: %s", stockData.ToString())
}

// =====================================================================================================================

func notifyPoolService(uglyStockEvt *messages.StockEvent) error {
	if bytes, err := json.Marshal(uglyStockEvt); err != nil {
		return err
	} else {
		if err := g.NsqProducer.Publish(g.PoolFeedsTopic, bytes); err != nil {
			redislogger.Errorf("notifyPoolService, failed to write to nsq, err: %v", err)
			return err
		}
	}
	return nil
}

func notifyEvtService(uglyStockEvt *messages.StockEvent, stockData *types.StockData) error {
	if bytes, err := json.Marshal(uglyStockEvt); err != nil {
		return err
	} else {
		if err := g.NsqProducer.Publish(g.EvtFeedsTopic, bytes); err != nil {
			redislogger.Errorf("notifyEvtService, failed to write to nsq, err: %v", err)
			return err
		}
	}
	// notify xgb backend about this Yidong event
	go func() {
		callYiDongApi(uglyStockEvt, stockData)
	}()
	return nil
}

func NiceStockEvtToUglyStockEvt(niceStockEvt *types.NiceStockEvt) *messages.StockEvent {
	return &messages.StockEvent{
		Symbol:          niceStockEvt.Sym,
		Name:            niceStockEvt.Name,
		Event:           niceStockEvt.PriceAction,
		Timestamp:       niceStockEvt.TradeAt,
		ChangePCT:       niceStockEvt.PriceChgRate,
		Price:           niceStockEvt.Price,
		MTM:             niceStockEvt.MTM,
		VolumeBiasRatio: niceStockEvt.VolBiasRatio,
		TickVol:         float64(niceStockEvt.TickVol),
		TradeStatus:     niceStockEvt.TradeStatus,
	}
}

func NiceStockEvtToCacheStockEvt(niceStockEvt *types.NiceStockEvt) *types.CacheStockEvt {
	return &types.CacheStockEvt{
		PriceAction: niceStockEvt.PriceAction,
		TradeAt:     niceStockEvt.TradeAt,
		PublishedAt: time.Now().Unix(),
	}
}

func CreateNiceStockEvtFromStockData(stockData *types.StockData) *types.NiceStockEvt {
	return &types.NiceStockEvt{
		Sym:          stockData.Sym,
		Name:         stockData.Name,
		Price:        stockData.Price,
		PriceChgRate: stockData.PriceChgRate,
		TickVol:      stockData.TickVol,
		MTM:          stockData.MTM,
		VolBiasRatio: stockData.VolBiasRatio,
		TradeAt:      stockData.TradeAt,
		TradeStatus:  stockData.TradeStatus,
		PublishedAt:  time.Now().Unix(),
	}
}

// =====================================================================================================================

type YidongMsgReq struct {
	Content     string
	TypeId      string
	StockSymbol string
	Secret      string
}

type YiDongMsgResp struct {
	ErrCode int
	ErrMsg  string
}

func callYiDongApi(uglyStockEvt *messages.StockEvent, stockData *types.StockData) error {
	var yidongMsgReq YidongMsgReq
	var priceActionStr string
	var typeId int64
	if uglyStockEvt.Event == flashcommon.EnumPriceActionLockLimitUp {
		priceActionStr = "封涨停板"
		typeId = 1
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionComeOffLimitUp {
		priceActionStr = "涨停板打开"
		typeId = 2
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionFreshStockComeOffLimitUp {
		priceActionStr = "新股开板"
		typeId = 3
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionFreshStockLockLimitUpAgain {
		priceActionStr = "新股开板回封"
		typeId = 4
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionApproachLimitUp {
		priceActionStr = "逼近涨停"
		typeId = 5
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionNearLimitUpBreach {
		priceActionStr = "即将打开涨停"
		typeId = 6
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionSurge {
		priceActionStr = "大幅拉升"
		typeId = 7
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionLockLimitDown {
		priceActionStr = "封跌停板"
		typeId = 8
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionComeOffLimitDown {
		priceActionStr = "跌停板打开"
		typeId = 9
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionApproachLimitDown {
		priceActionStr = "逼近跌停"
		typeId = 10
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionNearLimitDownBreach {
		priceActionStr = "即将打开跌停"
		typeId = 11
	} else if uglyStockEvt.Event == flashcommon.EnumPriceActionPlummet {
		priceActionStr = "快速跳水"
		typeId = 12
	} else {
		// priceActionStr = "N/A"
		// typeId = 0
		redislogger.Errorf("callYiDongApi, unrecognized event: %d", uglyStockEvt.Event)
		return nil
	}
	var priceChangeStr string
	if stockData.PriceChgRate > 0 {
		priceChangeStr = "涨幅"
	} else if stockData.PriceChgRate < 0 {
		priceChangeStr = "跌幅"
	}
	yidongMsgReq.Content = fmt.Sprintf("%s%s，最新价%.2f，%s%.2f%%，换手率%.2f%%", stockData.Name, priceActionStr,
		stockData.Price, priceChangeStr, stockData.PriceChgRate, stockData.TurnoverRatio)
	yidongMsgReq.TypeId = strconv.Itoa(int(typeId))
	yidongMsgReq.StockSymbol = uglyStockEvt.Symbol
	yidongMsgReq.Secret = g.SysConf.StockYidong.Secret
	jsonBytes, err := json.Marshal(yidongMsgReq)
	if err != nil {
		redislogger.Errorf("callYiDongApi, json marshal err: %v", err)
		return err
	}

	redislogger.Printf("Start calling yidong url %s", g.SysConf.StockYidong.CreateMsgUrl)
	resp, err := http.Post(g.SysConf.StockYidong.CreateMsgUrl, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		redislogger.Errorf("callYiDongApi, failed to POST data, err: %v", err)
		return err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		redislogger.Errorf("callYiDongApi, failed to read resp body, err: %v", err)
		return err
	}
	var yidongMsgResp YiDongMsgResp
	if err := json.Unmarshal(bodyBytes, &yidongMsgResp); err != nil {
		redislogger.Errorf("callYiDongApi, json unmarhal err: %v", err)
		return err
	}
	if yidongMsgResp.ErrCode > 0 {
		redislogger.Errorf("callYiDongApi, yidong resp indicates error, msg: %s", yidongMsgResp.ErrMsg)
	}
	return nil
}
