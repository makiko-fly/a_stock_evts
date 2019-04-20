package types

import (
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
)

// I call it "NiceStockEvt" because messages.StockEvt is terribly named
type NiceStockEvt struct {
	Sym          string
	Name         string
	Price        float64
	PriceChgRate float64
	TickVol      int64
	MTM          float64
	VolBiasRatio float64
	PriceAction  flashcommon.EnumPriceAction
	TradeStatus  string
	TradeAt      int64
	PublishedAt  int64
}
