package business

import (
	"context"
	"sync"
	"time"

	"gitlab.wallstcn.com/baoer/flash/flashcommon/protos/common"
	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
)

var symToD5VolMap = map[string]int64{}
var freshStockMap = map[string]string{} // sym to ipo date map
var tradingStartMap = map[string]bool{}
var tradingStartLock sync.RWMutex
var tradingStartKey = "TradingStart"

func DailyInit() error {
	redislogger.PrintfToSlot(1, "===> DailyInit started at %s", time.Now())

	if err := g.MinutePriceCache.ClearAll(); err != nil {
		redislogger.ErrorfToSlot(1, "DailyInit, failed to clear minute price cache, err: %v", err)
	}
	if err := g.StockStatCache.ClearAll(); err != nil {
		redislogger.Errorf("DailyInit, failed to clear stock stat cache, err: %v", err)
	}
	if err := g.PremarketCache.ClearAll(); err != nil {
		redislogger.ErrorfToSlot(1, "DailyInit, failed to clear premarket cache, err: %v", err)
	}
	if err := g.PremarketCache.SetTradeStartEvtSent(false); err != nil {
		redislogger.ErrorfToSlot(1, "DailyInit, failed to set TradeStartEvtSent to false, err: %v", err)
	} else {
		tradingStartLock.Lock()
		tradingStartMap[tradingStartKey] = false
		tradingStartLock.Unlock()
	}
	if err := g.StockEvtCache.ClearAll(); err != nil {
		redislogger.ErrorfToSlot(1, "DailyInit, failed to clear stock event cache, err: %v", err)
	}

	FetchAndPopulateD5AvgVolMap()
	FetchTodayFreshStocks()
	PopulateFreshStocksMap()

	redislogger.PrintfToSlot(1, "===> DailyInit finished at %v", time.Now())
	return nil
}

func DailyCleanup() error {
	redislogger.PrintfToSlot(1, "===> DailyCleanup started at %s", time.Now())

	if err := g.FreshStockCache.RemMarkedStocks(); err != nil {
		redislogger.ErrorfToSlot(1, "DailyCleanup, failed to remove marked stocks from fresh stock hash, err: %v", err)
	}

	redislogger.PrintfToSlot(1, "===> DailyCleanup finished at %s", time.Now())
	return nil
}

func FetchAndPopulateD5AvgVolMap() {
	// 获取所有 A 股的前五日平均成交量
	intReq := &common.IntRequest{Number: 5}
	d5VolStartAt := time.Now()
	redislogger.ErrorfToSlot(1, "FetchD5AvgVolMap, about to call rpc external.AStocksTurnoverVolumeAvg")

	ctx, _ := context.WithTimeout(context.Background(), time.Minute*5)
	if ret, err := g.ExternalClient.AStocksTurnoverVolumeAvg(ctx, intReq); err != nil {
		redislogger.ErrorfToSlot(1, "FetchD5AvgVolMap, failed to call rpc external.AStocksTurnoverVolumeAvg, err: %v", err)
	} else {
		redislogger.PrintfToSlot(1, "FetchD5AvgVolMap, got 5-day average volume map, entries: %d, took %v",
			len(ret.Volumes), time.Since(d5VolStartAt))
		symToInt64Map := map[string]int64{}
		var i = 0
		for key, val := range ret.Volumes {
			if i < 10 {
				i++
				redislogger.PrintfToSlot(1, "random entry %d, key: %v, val: %v", i, key, val)
			}
			symToInt64Map[key] = int64(val)
		}
		if err := g.D5AvgVolCache.Set(symToInt64Map); err != nil {
			redislogger.ErrorfToSlot(1, "Failed to set d5AvgVolCache, err: %v", err)
		} else {
			symToD5VolMap = symToInt64Map
		}
	}
}

func FetchTodayFreshStocks() {
	// get today's new stock
	todayStr := time.Now().Format("2006-01-02")
	listedDateReq := &common.StringRequest{Text: todayStr}
	if ret, err := g.ExternalClient.AStocksByListedDate(context.Background(), listedDateReq); err != nil {
		redislogger.ErrorfToSlot(1, "FetchTodayFreshStocks, failed to call rpc external.AStocksByListedDate, err: %v", err)
	} else {
		redislogger.PrintfToSlot(1, "FetchTodayFreshStocks, got today's fresh stocks from rpc: %v", ret.Stocks)
		for _, sym := range ret.Stocks {
			if err := g.FreshStockCache.Add(sym, todayStr); err != nil {
				redislogger.ErrorfToSlot(1, "FetchTodayFreshStocks, failed to add new stock to cache, err: %v", err)
			}
		}
	}
}

func PopulateD5AvgVolMap() error {
	if tmpMap, err := g.D5AvgVolCache.GetAsMap(); err != nil {
		redislogger.ErrorfToSlot(1, "PopulateD5AvgVolMap, failed to read d5AvgVolMap from cache, err: %v", err)
		return err
	} else {
		redislogger.PrintfToSlot(1, "PopulateD5AvgVolMap, got d5AvgVolMap from redis, entry num: %d", len(tmpMap))
		var i = 0
		for key, val := range tmpMap {
			if i < 10 {
				i++
				redislogger.PrintfToSlot(1, "random entry %d, key: %v, val: %v", i, key, val)
			}
		}
		// redislogger.PrintfToSlot(1, "PopulateD5AvgVolMap, got d5AvgVolMap from redis, value: %v", tmpMap)
		symToD5VolMap = tmpMap
		return nil
	}
}

func PopulateFreshStocksMap() error {
	if tmpMap, err := g.FreshStockCache.GetAllAsMap(); err != nil {
		redislogger.ErrorfToSlot(1, "PopulateFreshStocksMap, failed to read fresh stock map from cache, err: %v", err)
		return err
	} else {
		freshStockMap = tmpMap
		redislogger.PrintfToSlot(1, "PopulateFreshStocksMap, got fesh stock map from redis: %v", freshStockMap)
		return nil
	}
}

func PopulateTradingStartMap() error {
	if tradeStart, err := g.PremarketCache.GetTradeStartEvtSent(); err != nil {
		redislogger.ErrorfToSlot(1, "PopulateTradingStartMap, failed to read tradingStart value from cache, err: %v", err)
		// use machine time as a rough measure
		now := time.Now()
		var started = false
		ocallEndTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 25, 0, 0, flashstd.TZShanghai())
		if now.After(ocallEndTime) {
			started = true
		}
		tradingStartLock.Lock()
		tradingStartMap[tradingStartKey] = started
		tradingStartLock.Unlock()
		return err
	} else {
		tradingStartLock.Lock()
		tradingStartMap[tradingStartKey] = tradeStart
		tradingStartLock.Unlock()
		return nil
	}
}
