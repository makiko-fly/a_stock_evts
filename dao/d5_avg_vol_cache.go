package dao

import (
	// "gitlab.wallstcn.com/baoer/flash/flashstd"
	// "gitlab.wallstcn.com/baoer/flash/flashtick/enums"
	"errors"
	"fmt"
	"gopkg.in/redis.v5"
	"strconv"
	"strings"
)

type D5AvgVolCacheDAO struct {
	rc *redis.Client
}

func NewD5AvgVolCacheDAO(rc *redis.Client) *D5AvgVolCacheDAO {
	return &D5AvgVolCacheDAO{
		rc: rc,
	}
}

func (self *D5AvgVolCacheDAO) Set(dataMap map[string]int64) error {
	strArr := []string{}
	for sym, vol := range dataMap {
		strArr = append(strArr, fmt.Sprintf("%s:%d", sym, vol))
	}
	finalStr := strings.Join(strArr, ",")
	if _, err := self.rc.Set(KeyD5AvgVol, finalStr, -1).Result(); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *D5AvgVolCacheDAO) GetAsMap() (map[string]int64, error) {
	if wholeStr, err := self.rc.Get(KeyD5AvgVol).Result(); err != nil {
		return nil, err
	} else if strings.TrimSpace(wholeStr) == "" {
		return map[string]int64{}, nil
	} else {
		retMap := map[string]int64{}
		strArr := strings.Split(wholeStr, ",")
		for _, str := range strArr {
			tmpArr := strings.Split(str, ":")
			if len(tmpArr) < 2 {
				return nil, errors.New("D5AvgVolCache, GetAsMap, entry doesn't contain colon: " + str)
			}
			sym := tmpArr[0]
			vol, err := strconv.ParseInt(tmpArr[1], 10, 64)
			if err != nil {
				return nil, err
			}
			retMap[sym] = vol
		}
		return retMap, nil
	}
}
