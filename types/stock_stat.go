package types

import (
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
)

type StockStat struct {
	UpDownStatus        flashcommon.EnumUpDownStatus
	UpDownStatusSince   int64
	LastPrice           float64
	LastPriceChagRate   float64
	LastTradeAt         int64
	LastTurnoverVol     int64
	CurMinuteIdx        int64
	CurMinuteVol        int64
	LastMinuteVol       int64
	ApproachedLimitUp   bool
	ApproachedLimitDown bool
}
