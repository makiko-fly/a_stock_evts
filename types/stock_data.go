package types

import (
	"fmt"
	"time"
)

// our own stock data type, avoid some naming issues of TickData
type StockData struct {
	Sym           string
	Name          string
	TradeStatus   string
	Price         float64
	PriceChgRate  float64
	TradeAt       int64
	PreClosePrice float64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	TurnoverVol   int64
	TurnoverVal   float64
	TurnoverRatio float64 // 换手率
	SellPrice1    float64
	SellVol1      int64
	SellPrice2    float64
	SellVol2      int64
	SellPrice3    float64
	SellVol3      int64
	SellPrice4    float64
	SellVol4      int64
	SellPrice5    float64
	SellVol5      int64
	BuyPrice1     float64
	BuyVol1       int64
	BuyPrice2     float64
	BuyVol2       int64
	BuyPrice3     float64
	BuyVol3       int64
	BuyPrice4     float64
	BuyVol4       int64
	BuyPrice5     float64
	BuyVol5       int64
	MTM           float64
	VolBiasRatio  float64
	TickVol       int64 // 当前 tick 的交易量
	TrackId       int32
}

func (self *StockData) ToString() string {
	tradeTime := time.Unix(self.TradeAt, 0)
	return fmt.Sprintf("Sym:%s, Name:%s, TradeStatus:%s, Price:%.2f, PriceChgRate:%.2f, TradeAt:%v, PreClosePrice:%.2f, "+
		"Open:%.2f, High:%.2f, Low:%.2f, Close:%.2f, TurnoverVol:%d, TurnoverVal:%.2f, TurnoverRatio:%.2f, "+
		"SellPrice1:%.2f, SellVol1:%d, "+
		"SellPrice2:%.2f, SellVol2:%d, SellPrice3:%.2f, SellVol3:%d, SellPrice4:%.2f, SellVol4:%d, SellPrice5:%.2f, "+
		"SellVol5:%d, BuyPrice1:%.2f, BuyVol1:%d, BuyPrice2:%.2f, BuyVol2:%d, BuyPrice3:%.2f, BuyVol3:%d, "+
		"BuyPrice4:%.2f, BuyVol4:%d, BuyPrice5:%.2f, BuyVol5:%d, MTM:%.2f, VolBiasRatio:%.2f, TickVol:%d", self.Sym,
		self.Name, self.TradeStatus, self.Price, self.PriceChgRate, tradeTime, self.PreClosePrice, self.Open,
		self.High, self.Low, self.Close, self.TurnoverVol, self.TurnoverVal, self.TurnoverRatio,
		self.SellPrice1, self.SellVol1,
		self.SellPrice2, self.SellVol2, self.SellPrice3, self.SellVol3, self.SellPrice4, self.SellVol4,
		self.SellPrice5, self.SellVol5, self.BuyPrice1, self.BuyVol1, self.BuyPrice2, self.BuyVol2,
		self.BuyPrice3, self.BuyVol3, self.BuyPrice4, self.BuyVol4, self.BuyPrice5, self.BuyVol5, self.MTM,
		self.VolBiasRatio, self.TickVol)
}
