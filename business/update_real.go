package business

import (
	"time"

	"gitlab.wallstcn.com/baoer/flash/flashcommon/messages"
	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gitlab.wallstcn.com/baoer/flash/flashstd/json"
	"gitlab.wallstcn.com/baoer/flash/flashtick/g"
	"gitlab.wallstcn.com/baoer/flash/flashtick/redislogger"
)

func UpdateRealAsync(updatedTickData *messages.TickData) {
	if g.NsqProducer == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				flashstd.Errorf("Recovered from UpdateRealAsync: %v", r)
			}
		}()

		startTime := time.Now()
		if bytes, err := json.Marshal(updatedTickData); err != nil {
			flashstd.WithError(err).Error("Error in marshal tick data")
		} else {
			if err := g.NsqProducer.Publish(g.RealFeedsTopic, bytes); err != nil {
				flashstd.Errorf("UpdateRealAsync, failed to write nsq, err: %v", err)
			}
		}
		redislogger.Trackf(updatedTickData.TrackId, "Took %v to write stock price to nsq", time.Since(startTime))
	}()
}
