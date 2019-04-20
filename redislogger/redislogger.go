package redislogger

import (
	"fmt"
	"os"
	"time"

	"gitlab.wallstcn.com/baoer/flash/flashstd"
	"gopkg.in/redis.v5"
)

var redisClient *redis.Client

const KeyLogList string = "custom:log:list"
const MaxListSize = 20000

func InitWithRedisClient(client *redis.Client) {
	redisClient = client
}

func getKey(slot int) string {
	if slot <= 0 {
		return KeyLogList
	}
	return fmt.Sprintf("%s:%d", KeyLogList, slot)
}

func println(slot int, trackId int32, line string) error {
	if redisClient == nil {
		return nil
	}

	var hostName string
	if tmpStr, err := os.Hostname(); err != nil {

	} else {
		// get last 5 characters of host
		if len(tmpStr) > 5 {
			hostName = tmpStr[len(tmpStr)-5:]
		} else {
			hostName = tmpStr
		}
	}

	timeStr := time.Now().Format("2006-01-02 15:04:05.000")
	var finalStr string
	if trackId > 0 {
		finalStr = fmt.Sprintf("[%s] [%s] [%d] %s", timeStr, hostName, trackId, line)
	} else {
		finalStr = fmt.Sprintf("[%s] [%s] %s", timeStr, hostName, line)
	}

	// print to console
	fmt.Println(finalStr)

	if newSize, err := redisClient.LPush(getKey(slot), finalStr).Result(); err != nil {
		return err
	} else if newSize > MaxListSize {
		redisClient.RPop(getKey(slot))
	}
	return nil
}

func Printf(format string, v ...interface{}) {
	line := fmt.Sprintf(format, v...)
	println(0, 0, line)
}

func Trackf(trackId int32, format string, v ...interface{}) {
	if trackId == 0 {
		return
	}
	line := fmt.Sprintf(format, v...)
	println(0, trackId, line)
}

// print to redis as well as flashstd error
func Errorf(format string, v ...interface{}) {
	Printf(format, v...)
	flashstd.Errorf(format, v...)
}

func PrintfToSlot(slot int, format string, v ...interface{}) {
	line := fmt.Sprintf(format, v...)
	println(slot, 0, line)
}

func ErrorfToSlot(slot int, format string, v ...interface{}) {
	PrintfToSlot(slot, format, v...)
	flashstd.Errorf(format, v...)

}
