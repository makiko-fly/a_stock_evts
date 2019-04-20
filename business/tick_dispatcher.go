package business

import (
	// "bytes"
	// "fmt"
	"github.com/nsqio/go-nsq"
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
	"gitlab.wallstcn.com/baoer/flash/flashcommon/messages"
	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gitlab.wallstcn.com/baoer/flash/flashstd/json"

	// "gitlab.wallstcn.com/baoer/flash/flashtick/dao"
	// "gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
	// "strconv"
	// "math"
	// "math/rand"
	"strconv"
	// "strings"
	"time"
)

// var (
// 	securitiesStock = []byte(`"securities_type":"stock"`)
// )

type TickDispather struct {
	ConsumerNum int
	Consumers   []*TickConsumer
}

func NewTickDispather(consumerNum int, bufferSize int) *TickDispather {
	consumers := make([]*TickConsumer, 0, consumerNum)
	for i := 0; i < consumerNum; i++ {
		consumer := NewTickConsumer(HandleTickData, bufferSize)
		consumer.StartAsync()
		consumers = append(consumers, consumer)
	}
	return &TickDispather{
		ConsumerNum: consumerNum,
		Consumers:   consumers,
	}
}

func (self *TickDispather) HandleMessage(message *nsq.Message) error {
	// if !bytes.Contains(message.Body, securitiesStock) {
	// 	return nil
	// }

	var tickData messages.TickData
	if err := json.Unmarshal(message.Body, &tickData); err != nil {
		redislogger.ErrorfToSlot(1, "HandleMessage, failed to unmarshal tick data: %s, err: %v", string(message.Body), err)
		return err
	}

	sym := tickData.ProdCode
	tradeTime := time.Unix(tickData.LastAt, 0)

	// 过滤 B 股，代码开头为 9 的是在上交所交易的，开头为 2 的是在深交所交易的
	if sym[0] == '9' || sym[0] == '2' {
		return nil
	}

	// 只看交易中的股票
	if !flashstd.IsTradeTime(tradeTime) {
		return nil
	}

	// 不看停牌数据
	if tickData.TradeStatus == "HALT" {
		// 停牌股的数据实时行情那里也需要
		UpdateRealAsync(&tickData)
		return nil
	}

	// 下午两点后不再看竞价集合的数据
	if tradeTime.Hour() >= 14 && tickData.TradeStatus == "OCALL" {
		return nil
	}

	// 若买一和卖一都为 0，则为无效数据
	if tickData.BuyPrice1 == 0 && tickData.SellPrice1 == 0 {
		return nil
	}

	// 竞价期，深交所不传最新价，要把卖一价作为最新价
	if tradeTime.Hour() == 9 && 15 <= tradeTime.Minute() && tradeTime.Minute() <= 25 &&
		tickData.TradeStatus == flashcommon.TradeStatusOCALL {
		if sym[len(sym)-2:] == "SZ" {
			tickData.LastPx = tickData.SellPrice1
		}
	}

	// dispatch
	lastTwoDigitStr := string(sym[len(sym)-5 : len(sym)-3])
	lastTwoDigits, err := strconv.Atoi(lastTwoDigitStr)
	if err != nil {
		return err
	}
	idx := lastTwoDigits % self.ConsumerNum
	self.Consumers[idx].Put(&tickData)

	return nil
}
