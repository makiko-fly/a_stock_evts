package types

type MinPriceItem struct {
	MinIdx    int64 // number of miniutes passed since opening
	Timestamp int64 // actual timestamp of the tick
	Price     float64
}
