package business

import (
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
	"gitlab.wallstcn.com/baoer/flash/flashcommon/messages"
	"gitlab.wallstcn.com/baoer/flash/flashstd"

	// "gitlab.wallstcn.com/baoer/flash/flashtick/dao"
	"gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"

	// "strconv"
	"math"
	"math/rand"
	"strings"
	"time"
)

var lastPrintAt int64

type TickConsumer struct {
	Chann   chan *messages.TickData
	Handler func(tickData *messages.TickData) error
}

func NewTickConsumer(handler func(tickData *messages.TickData) error, bufferSize int) *TickConsumer {
	return &TickConsumer{
		Chann:   make(chan *messages.TickData, bufferSize),
		Handler: handler,
	}
}

func (self *TickConsumer) Put(tickData *messages.TickData) {
	self.Chann <- tickData
}

func (self *TickConsumer) StartAsync() {
	go func() {
		for tickData := range self.Chann {
			self.Handler(tickData)
		}
	}()
}

func HandleTickData(tickData *messages.TickData) error {
	startTime := time.Now()
	tmpTime := startTime

	var trackId int32 = 0
	if startTime.Unix()-lastPrintAt > 50 {
		lastPrintAt = startTime.Unix()
		trackId = rand.Int31()
		tickData.TrackId = trackId
		redislogger.Trackf(trackId, "Start handling tick data: %v", tickData)
	}

	// PxChangeRate is not calculated before-hand for us.
	tickData.PxChangeRate = (tickData.LastPx/tickData.PreclosePx - 1.0) * 100

	stockData := TickDataToStockData(tickData)
	tradeTime := time.Unix(stockData.TradeAt, 0)
	isSt := isSt(stockData.Name)
	passedMinutes := flashstd.PassedTradingMinutes(stockData.TradeAt)
	var mtm, volBiasRatio float64

	// 9 点 15 分到 25 分为早盘集合竞价，期间产生的事件不计入正式交易
	if tradeTime.Hour() == 9 && 15 <= tradeTime.Minute() && tradeTime.Minute() <= 25 &&
		stockData.TradeStatus == flashcommon.TradeStatusOCALL {
		processPremarketTick(isSt, stockData)
		UpdateRealAsync(tickData)
		return nil
	}

	// if we get here, we have passed premarket stage.
	// 9 点 25 分会产生第一笔正式数据，上交所的 tick 状态为 TRADE，交易时间为 9 点 25 分 00 秒，
	// 深交所的状态为 BREAK，交易时间为 9 点 25 分前几秒，如 03，04 秒
	if tradeTime.Hour() == 9 && tradeTime.Minute() == 25 {
		// // 打印 9 点 25 分所有 tick
		// if bytes, err := flashstd.Marshal(stockData); err != nil {
		// 	return err
		// } else {
		// 	redislogger.PrintfToSlot(1, "tick data at 9:25, data: %v", string(bytes))
		// }

		// 收到第一个 status 为 TRADE 或者 BREAK 的 tick 后，发出一个 event，通知 pool 清空涨停跌停
		// 这里用 stock event 充当了别的用处，不是一个好的做法
		if stockData.TradeStatus == flashcommon.TradeStatusTRADE || stockData.TradeStatus == flashcommon.TradeStatusBREAK {
			tradingStartLock.Lock()
			if tradingStarted, found := tradingStartMap[tradingStartKey]; found {
				if tradingStarted {
					// do nothing
					tradingStartLock.Unlock()
				} else {
					uglyStockEvt := &messages.StockEvent{Event: flashcommon.EventTradingStart, Timestamp: time.Now().Unix()}
					notifyPoolService(uglyStockEvt)
					redislogger.PrintfToSlot(1, "Handle tick, sent TradingStart event to pool")
					tradingStartMap[tradingStartKey] = true
					tradingStartLock.Unlock()

					if err := g.PremarketCache.SetTradeStartEvtSent(true); err != nil {
						redislogger.ErrorfToSlot(1, "Handle tick, failed to set TradeStartEvtSent to true, err: %v", err)
					}
				}
			} else {
				// won't happen, do nothing
				tradingStartLock.Unlock()
			}
		}
	}

	// 获取上个 tick 的状态
	stockStat, err := g.StockStatCache.GetBySym(stockData.Sym)
	if err != nil {
		redislogger.ErrorfToSlot(1, "HandleTickData, failed to get stock stat from redis, sym: %s err: %v", stockData.Sym, err)
	} else if stockStat == nil { // create default stock stat
		stockStat = &types.StockStat{UpDownStatus: flashcommon.EnumUpDownStatusNormal, LastPriceChagRate: stockData.PriceChgRate}
		if isNewStock(stockData.Sym) { // 新股初始 tick 为涨停
			stockStat.UpDownStatus = flashcommon.EnumUpDownStatusLockedLimitUp
			stockStat.UpDownStatusSince = stockData.TradeAt
		}
	} else {
		// 检查是否和上个 tick 重复
		if roundDecimal(stockData.Price) == roundDecimal(stockStat.LastPrice) && stockData.TradeAt == stockStat.LastTradeAt &&
			stockData.TurnoverVol == stockStat.LastTurnoverVol {
			return nil
		}
		// 计算当前 tick 交易量
		stockData.TickVol = stockData.TurnoverVol - stockStat.LastTurnoverVol
		// 如果三点收盘后，计算出某个 tick 的交易量为 0，则忽略，该措施是为了解决深交所发好几个 tick，且交易时间都不一样的问题
		if tradeTime.Hour() >= 15 && stockData.TickVol == 0 {
			return nil
		}
	}

	redislogger.Trackf(trackId, "Took %v to get stock stat, since start: %v", time.Since(tmpTime), time.Since(startTime))
	tmpTime = time.Now()

	// 计算涨速和量比
	// 获取前五分钟的分钟价格数据，并尝试更新当前分钟的价格数据
	// 9 点 25 分最后一笔正式价格算作 9 点 30 分这一分钟的价格
	minutePriceItems, err := g.MinutePriceCache.LatestFiveItems(stockData.Sym)
	if err != nil {
		redislogger.ErrorfToSlot(1, "HandleTickData, failed to get latest five minute prices from redis, err: %v", err)
	} else {
		if len(minutePriceItems) == 0 || minutePriceItems[0].MinIdx < passedMinutes {
			g.MinutePriceCache.Prepend(stockData.Sym, passedMinutes, stockData.TradeAt, stockData.Price)
		}
		mtm = calcMTM(stockData, passedMinutes, minutePriceItems)
	}
	volBiasRatio = calVolBiasRatio(stockData.Name, stockData.Sym, stockData.TurnoverVol, passedMinutes)
	tickData.MTM = mtm
	stockData.MTM = mtm
	tickData.VolumeBiasRatio = volBiasRatio
	stockData.VolBiasRatio = volBiasRatio

	redislogger.Trackf(trackId, "Took %v to get and set minute price data, since start: %v", time.Since(tmpTime), time.Since(startTime))
	tmpTime = time.Now()

	processStockTick(isSt, stockData, stockStat, passedMinutes, mtm)
	redislogger.Trackf(trackId, "Took %v to processStockTick, since start: %v", time.Since(tmpTime), time.Since(startTime))

	UpdateRealAsync(tickData)
	redislogger.Trackf(trackId, "Finished all work of this tick, since start: %v", time.Since(startTime))
	return nil
}

func processStockTick(isSt bool, stockData *types.StockData, stockStat *types.StockStat, passedMinutes int64,
	mtm float64) {

	sym := stockData.Sym
	price := stockData.Price
	preClosePrice := stockData.PreClosePrice
	priceChgRate := stockData.PriceChgRate
	tradeAt := stockData.TradeAt

	// 检查股票异动
	newUpDownStatus := stockStat.UpDownStatus
	var approachedLimitUp bool = false
	var approachedLimitDown bool = false
	limitUpPrice := calcLimitUpPrice(isSt, preClosePrice)
	limitDownPrice := calcLimitDownPrice(isSt, preClosePrice)
	upThresholdPrice := calcUpThresholdPrice(preClosePrice, limitUpPrice)
	downThresholdPrice := calcDownThresholdPrice(preClosePrice, limitDownPrice)
	if stockStat.UpDownStatus == flashcommon.EnumUpDownStatusLockedLimitUp { // 从涨停板可以进入打开涨停板或者即将打开涨停板状态
		if roundDecimal(stockData.BuyPrice1) >= roundDecimal(limitUpPrice) { // 继续封板，检查是否即将打开涨停状态
			// 封板时间大于 1 分钟才能算是比较稳定
			if flashstd.PassedTradingSeconds(tradeAt)-flashstd.PassedTradingSeconds(stockStat.UpDownStatusSince) > 60 {
				accumVol := getAccumVal(stockStat, passedMinutes)
				if accumVol > 0 && float64(accumVol)/float64(stockData.BuyVol1) > 0.10 { //（当前分钟内成交量+上1分钟成交量）占买一量的比例大于10%
					PubAboutToComeOffLimitUpEvt(stockData, accumVol)
				}
			}
		} else { // 打开涨停状态
			newUpDownStatus = flashcommon.EnumUpDownStatusNormal
			PubComeOffLimitUpEvt(stockData, limitUpPrice)
		}
	} else if stockStat.UpDownStatus == flashcommon.EnumUpDownStatusLockedLimitDown { // 从跌停板可以进入打开跌停板或者即将打开跌停板
		if roundDecimal(stockData.SellPrice1) <= roundDecimal(limitDownPrice) { // 继续跌停，检查是否即将打开跌停状态
			// 封板时间大于 1 分钟才能算是比较稳定
			if flashstd.PassedTradingSeconds(tradeAt)-flashstd.PassedTradingSeconds(stockStat.UpDownStatusSince) > 60 {
				accumVol := getAccumVal(stockStat, passedMinutes)
				if accumVol > 0 && float64(accumVol)/float64(stockData.SellVol1) > 0.10 { //（当前分钟内成交量+上1分钟成交量）占买一量的比例大于10%
					PubAboutToComeOffLimitDownEvt(stockData, accumVol)
				}
			}
		} else { // 打开跌停状态
			newUpDownStatus = flashcommon.EnumUpDownStatusNormal
			PubComeOffLimitDownEvt(stockData, limitDownPrice)
		}
	} else { // 该股票之前处于普通状态
		// 检查封涨停板和封跌停板
		if roundDecimal(price) >= roundDecimal(limitUpPrice) &&
			roundDecimal(stockData.BuyPrice1) >= roundDecimal(limitUpPrice) { // 封涨停板
			newUpDownStatus = flashcommon.EnumUpDownStatusLockedLimitUp
			PubLockLimitUpEvt(stockData, limitUpPrice)
		} else if roundDecimal(price) <= roundDecimal(limitDownPrice) &&
			roundDecimal(stockData.SellPrice1) <= roundDecimal(limitDownPrice) { // 封跌停板
			newUpDownStatus = flashcommon.EnumUpDownStatusLockedLimitDown
			PubLockLimitDownEvt(stockData, limitDownPrice)
		} else if priceChgRate >= 0.0 { // 检查逼近涨停
			approachedLimitUp = checkApproachLimitUp(isSt, stockData, limitUpPrice, upThresholdPrice, stockStat)
			if !approachedLimitUp {
				// 检查大涨
				if priceChgRate > 3.0 && mtm > 3.0 {
					PubSurgeEvent(stockData)
				}
			}
		} else if priceChgRate < 0.0 { // 检查逼近跌停
			approachedLimitDown = checkApproachLimitDown(isSt, stockData, limitDownPrice, downThresholdPrice, stockStat)
			if !approachedLimitDown {
				// 检查大跌
				if priceChgRate < -3.0 && mtm < -3.0 {
					PubPlummetEvt(stockData)
				}
			}
		}
	}

	// update new stock stat
	var newStockStat types.StockStat
	newStockStat.LastPrice = price
	newStockStat.LastPriceChagRate = priceChgRate
	newStockStat.UpDownStatus = newUpDownStatus
	newStockStat.LastTurnoverVol = stockData.TurnoverVol
	newStockStat.LastTradeAt = stockData.TradeAt
	newStockStat.ApproachedLimitUp = stockStat.ApproachedLimitUp
	newStockStat.ApproachedLimitDown = stockStat.ApproachedLimitDown
	if newUpDownStatus != stockStat.UpDownStatus {
		newStockStat.UpDownStatusSince = tradeAt
	} else {
		newStockStat.UpDownStatusSince = stockStat.UpDownStatusSince
	}
	newStockStat.CurMinuteIdx = passedMinutes
	if passedMinutes == stockStat.CurMinuteIdx {
		newStockStat.CurMinuteVol += stockData.TickVol
		newStockStat.LastMinuteVol = stockStat.LastMinuteVol
	} else if passedMinutes == stockStat.CurMinuteIdx+1 {
		newStockStat.LastMinuteVol = stockStat.CurMinuteVol
		newStockStat.CurMinuteVol = stockData.TickVol
	}
	// update ApproachedLimitUp field
	if approachedLimitUp {
		newStockStat.ApproachedLimitUp = true
	} else if price < upThresholdPrice {
		newStockStat.ApproachedLimitUp = false
	}
	// update ApproachedLimitDown field
	if approachedLimitDown {
		newStockStat.ApproachedLimitDown = true
	} else if price > downThresholdPrice {
		newStockStat.ApproachedLimitDown = false
	}
	if err := g.StockStatCache.UpdateStat(sym, &newStockStat); err != nil {
		flashstd.Errorf("HandleMessage, failed to write new stock stat into redis, err: %v", err)
	}
}

func processPremarketTick(isSt bool, stockData *types.StockData) {
	// get last tick stat
	sym := stockData.Sym
	premarketStat, err := g.PremarketCache.GetBySym(sym)
	if err != nil {
		redislogger.ErrorfToSlot(1, "processPremarketTick, failed to get premarket stat from redis, sym: %s err: %v", sym, err)
	} else if premarketStat == nil {
		// 如果是新股，集合竞价期间的初始状态默认为涨停
		if isNewStock(sym) {
			premarketStat = &types.PremarketStat{UpDownStatus: flashcommon.EnumUpDownStatusLockedLimitUp}
		} else {
			premarketStat = &types.PremarketStat{UpDownStatus: flashcommon.EnumUpDownStatusNormal}
		}
	}

	limitUpPrice := calcLimitUpPrice(isSt, stockData.PreClosePrice)
	limitDownPrice := calcLimitDownPrice(isSt, stockData.PreClosePrice)
	var newUpDownStatus flashcommon.EnumUpDownStatus
	if roundDecimal(stockData.Price) >= roundDecimal(limitUpPrice) { // check limit up
		newUpDownStatus = flashcommon.EnumUpDownStatusLockedLimitUp
		if premarketStat.UpDownStatus != flashcommon.EnumUpDownStatusLockedLimitUp {
			PubLockLimitUpEvt(stockData, limitUpPrice)
		}
	} else if roundDecimal(stockData.Price) <= roundDecimal(limitDownPrice) {
		newUpDownStatus = flashcommon.EnumUpDownStatusLockedLimitDown
		if premarketStat.UpDownStatus != flashcommon.EnumUpDownStatusLockedLimitDown {
			PubLockLimitDownEvt(stockData, limitDownPrice)
		}
	} else {
		newUpDownStatus = flashcommon.EnumUpDownStatusNormal
		if premarketStat.UpDownStatus == flashcommon.EnumUpDownStatusLockedLimitUp {
			PubComeOffLimitUpEvt(stockData, limitUpPrice)
		} else if premarketStat.UpDownStatus == flashcommon.EnumUpDownStatusLockedLimitDown {
			PubComeOffLimitDownEvt(stockData, limitDownPrice)
		}
	}
	// update premarket stat cache
	if premarketStat.UpDownStatus != newUpDownStatus {
		g.PremarketCache.UpdateStat(sym, &types.PremarketStat{UpDownStatus: newUpDownStatus})
	}
}

// ======================================================================================================

// 涨速算法（9 点半开始计算）
// 最新价 / 5 分钟前的价格 - 1
// 不满 5 分钟的，分母是昨收
// 5 分钟前没成交的，往前取最靠近的
func calcMTM(stockData *types.StockData, passedMinutes int64, minutePriceItems []*types.MinPriceItem) float64 {
	// if stockData.PreClosePrice == 0 { // invalid stock data
	// 	flashstd.Warnf("Stock data's PreClosePrice field is 0, stockData: %v", stockData)
	// 	return 0.0
	// }
	if passedMinutes < 5 {
		return stockData.PriceChgRate // PriceChgRate value is within range -10 ~ 10, not -10% ~ 10%
	} else {
		if len(minutePriceItems) == 0 {
			flashstd.Warnf("Can't find trading records for symbol %s after 5 minutes, new stock data: %v",
				stockData.Sym, stockData)
			return 0.0
		} else {
			// start from latest to oldest
			for _, item := range minutePriceItems {
				if passedMinutes-item.MinIdx >= 5 {
					return ((stockData.Price / item.Price) - 1.0) * 100
				}
			}
			return 0.0
		}
	}
}

func calVolBiasRatio(stockName string, symbol string, volumnUntilNow int64, passedMinutes int64) float64 {
	if passedMinutes <= 0 {
		return 0.0
	}
	if d5Vol, found := symToD5VolMap[symbol]; !found {
		// redislogger.Printf("5 day average volumn not found for stock %s with symbol %s", stockName, symbol)
		return 0.0
	} else if d5Vol == 0 {
		return 0.0
	} else {
		// use passedMinutes plus 1 to reduce value spike
		return float64(volumnUntilNow*240) / float64((passedMinutes+1)*d5Vol)
	}
}

func calcLimitUpPrice(isSt bool, preClosePrice float64) float64 {
	if isSt {
		return preClosePrice * 1.05
	} else {
		return preClosePrice * 1.10
	}
}

func calcLimitDownPrice(isSt bool, preClosePrice float64) float64 {
	if isSt {
		return preClosePrice * 0.95
	} else {
		return preClosePrice * 0.90
	}
}

func calcUpThresholdPrice(preClosePrice, limitUpPrice float64) float64 {
	deduction := preClosePrice * 0.01
	if deduction < 0.02 {
		deduction = 0.02
	}
	return limitUpPrice - deduction
}

func calcDownThresholdPrice(preClosePrice, limitDownPrice float64) float64 {
	addition := preClosePrice * 0.01
	if addition < 0.02 {
		addition = 0.02
	}
	return limitDownPrice + addition
}

func isSt(stockName string) bool {
	// 代码开头为大写的 S 的股票也被认为是 ST 股
	return strings.Contains(stockName, "ST") || strings.Index(stockName, "S") == 0
}

func isNewStock(sym string) bool {
	if _, found := freshStockMap[sym]; found {
		return true
	} else {
		return false
	}
}

// round to 2 decimals
// multiply by 100, then round
func roundDecimal(price float64) int {
	return int(math.Round(price * 100))
}

func checkApproachLimitUp(isSt bool, stockData *types.StockData, limitUpPrice, upThresholdPrice float64,
	stockStat *types.StockStat) bool {
	if stockData.MTM > 1.0 { // 快速拉升型，前一笔涨幅小于 9，当前涨幅大于等于 9
		targetPriceChgRate := 9.0
		if isSt {
			targetPriceChgRate = 4.0
		}
		if stockStat.LastPriceChagRate < targetPriceChgRate && stockData.PriceChgRate >= targetPriceChgRate {
			PubApproachingLimitUpEvt(stockData, stockStat, "fast", targetPriceChgRate, 0.0)
			return true
		}
	} else { // 缓慢爬升型，边界区域：涨停价-max(昨收*1%,0.02)
		if stockStat.ApproachedLimitUp { // 之前的 tick 已经发出异动提醒了，不再提醒
			return false
		}
		if stockData.Price >= upThresholdPrice {
			var sellPrice float64
			if stockData.Price <= 2.0 {
				sellPrice = stockData.SellPrice1
			} else if 2.0 < stockData.Price && stockData.Price <= 5.0 {
				sellPrice = stockData.SellPrice3
			} else {
				sellPrice = stockData.SellPrice5
			}
			if roundDecimal(sellPrice) == 0 || roundDecimal(sellPrice) == roundDecimal(limitUpPrice) {
				PubApproachingLimitUpEvt(stockData, stockStat, "slowly", upThresholdPrice, limitUpPrice)
				return true
			}
		}
	}
	return false
}

func checkApproachLimitDown(isSt bool, stockData *types.StockData, limitDownPrice, downThresholdPrice float64,
	stockStat *types.StockStat) bool {
	if stockData.MTM < -1.0 { // 快速下跌型，跌速大于 1.0，前一笔跌幅小于 9，当前跌幅大于 9
		threshPrice := -9.0
		if isSt {
			threshPrice = -4.0
		}
		if stockStat.LastPriceChagRate > threshPrice && stockData.PriceChgRate <= threshPrice {
			PubApproachingLimitDownEvt(stockData, stockStat, "fast")
			return true
		}
	} else { // 缓慢下跌型，边界区域：跌停价+max(昨收*1%,0.02)
		if stockStat.ApproachedLimitDown { // 之前的 tick 已经发出异动提醒了，不再提醒
			return false
		}
		if stockData.Price <= downThresholdPrice {
			var buyPrice float64
			if stockData.Price <= 2.0 {
				buyPrice = stockData.BuyPrice1
			} else if 2.0 < stockData.Price && stockData.Price <= 5.0 {
				buyPrice = stockData.BuyPrice3
			} else {
				buyPrice = stockData.BuyPrice5
			}
			if roundDecimal(buyPrice) == 0 || roundDecimal(buyPrice) == roundDecimal(limitDownPrice) {
				PubApproachingLimitDownEvt(stockData, stockStat, "slowly")
				return true
			}
		}
	}
	return false
}

// 当前分钟内成交量 + 上 1 分钟成交量
func getAccumVal(stockStat *types.StockStat, passedMinutes int64) int64 {
	var accumVol int64 = 0
	if stockStat.CurMinuteIdx == passedMinutes { // 当前 tick 的时间戳在 stockStat 的当前分钟
		accumVol = stockStat.CurMinuteVol + stockStat.LastMinuteVol
	} else if stockStat.CurMinuteIdx == passedMinutes-1 { // 当前 tick 的时间戳在 stockStat 的下一分钟
		accumVol = stockStat.CurMinuteVol
	}
	return accumVol
}

func TickDataToStockData(tickData *messages.TickData) *types.StockData {
	return &types.StockData{
		Sym:           tickData.ProdCode,
		Name:          strings.TrimSpace(tickData.ProdName),
		TradeStatus:   tickData.TradeStatus,
		Price:         tickData.LastPx,
		PriceChgRate:  tickData.PxChangeRate,
		TradeAt:       tickData.LastAt,
		PreClosePrice: tickData.PreclosePx,
		Open:          tickData.OpenPx,
		High:          tickData.HighPx,
		Low:           tickData.LowPx,
		Close:         tickData.ClosePx,
		TurnoverVol:   tickData.TurnoverVolume,
		TurnoverVal:   tickData.TurnoverValue,
		TurnoverRatio: tickData.TurnoverRatio,
		SellPrice1:    tickData.SellPrice1,
		SellVol1:      tickData.SellVolume1,
		SellPrice2:    tickData.SellPrice2,
		SellVol2:      tickData.SellVolume2,
		SellPrice3:    tickData.SellPrice3,
		SellVol3:      tickData.SellVolume3,
		SellPrice4:    tickData.SellPrice4,
		SellVol4:      tickData.SellVolume4,
		SellPrice5:    tickData.SellPrice5,
		SellVol5:      tickData.SellVolume5,
		BuyPrice1:     tickData.BuyPrice1,
		BuyVol1:       tickData.BuyVolume1,
		BuyPrice2:     tickData.BuyPrice2,
		BuyVol2:       tickData.BuyVolume2,
		BuyPrice3:     tickData.BuyPrice3,
		BuyVol3:       tickData.BuyVolume3,
		BuyPrice4:     tickData.BuyPrice4,
		BuyVol4:       tickData.BuyVolume4,
		BuyPrice5:     tickData.BuyPrice5,
		BuyVol5:       tickData.BuyVolume5,
		TrackId:       tickData.TrackId,
	}
}
