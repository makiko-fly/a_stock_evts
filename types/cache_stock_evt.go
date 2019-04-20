package types

import (
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
)

type CacheStockEvt struct {
	PriceAction flashcommon.EnumPriceAction
	TradeAt     int64 // trade timestamp
	PublishedAt int64
}
