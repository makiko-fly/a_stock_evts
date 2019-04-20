package dao

const KeyMinPricePrefix string = "list:min:price:"
const MinPriceListMaxSize = 6

const KeyStockStatHash string = "hash:stock:stat"

const KeyPremarketStatHash string = "hash:premarket:stat"
const KeyTradeStartEvtSent string = "kv:tradeStartEvtSent"

const KeyStockEvtPrefix string = "list:stock:evt:"

const KeyFreshStockHash string = "hash:fresh:stock"
const KeyMarkedFreshStockHash string = "hash:fresh:stock:marked" // fresh stock comes off limit-up, mark for deletion after close.

const KeyD5AvgVol string = "kv:d5AvgVol"
