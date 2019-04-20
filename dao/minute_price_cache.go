package dao

import (
	"fmt"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
	"gopkg.in/redis.v5"
	"strconv"
	"strings"
)

type MinPriceCacheDAO struct {
	rc *redis.Client
}

func NewMinPriceCacheDAO(rc *redis.Client) *MinPriceCacheDAO {
	return &MinPriceCacheDAO{
		rc: rc,
	}
}

func (self *MinPriceCacheDAO) LatestFiveItems(sym string) ([]*types.MinPriceItem, error) {
	if strArr, err := self.rc.LRange(self.GetMinPriceCacheKey(sym), 0, 5).Result(); err != nil {
		return nil, err
	} else if len(strArr) == 0 {
		return make([]*types.MinPriceItem, 0), nil
	} else {
		retArr := make([]*types.MinPriceItem, 0)
		// str format: minIdx,timestamp,price
		for _, str := range strArr {
			tmpArr := strings.Split(str, ",")
			minIdx, _ := strconv.Atoi(tmpArr[0])
			timestamp, _ := strconv.ParseInt(tmpArr[1], 10, 64)
			price, _ := strconv.ParseFloat(tmpArr[2], 64)
			retArr = append(retArr, &types.MinPriceItem{
				MinIdx:    int64(minIdx),
				Timestamp: timestamp,
				Price:     price,
			})
		}
		return retArr, nil
	}
}

func (self *MinPriceCacheDAO) Prepend(sym string, minIdx int64, ts int64, price float64) error {
	strVal := fmt.Sprintf("%d,%d,%.2f", minIdx, ts, price)
	if length, err := self.rc.LPush(self.GetMinPriceCacheKey(sym), strVal).Result(); err != nil {
		return err
	} else if length > MinPriceListMaxSize {
		if _, err := self.rc.RPop(self.GetMinPriceCacheKey(sym)).Result(); err != nil {
			return err
		}
	}
	return nil
}

// clear all keys belonging to minute price cache
func (self *MinPriceCacheDAO) ClearAll() error {
	if keys, err := self.rc.Keys(fmt.Sprintf("%s*", KeyMinPricePrefix)).Result(); err != nil {
		return err
	} else {
		for _, key := range keys {
			if _, err := self.rc.Del(key).Result(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *MinPriceCacheDAO) GetMinPriceCacheKey(sym string) string {
	return KeyMinPricePrefix + sym
}
