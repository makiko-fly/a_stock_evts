package dao

import (
	"fmt"
	"gitlab.wallstcn.com/baoer/flash/flashcommon"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
	"gopkg.in/redis.v5"
	"strconv"
	"strings"
)

type StockEvtCacheDAO struct {
	rc *redis.Client
}

func NewStockEvtCacheDAO(rc *redis.Client) *StockEvtCacheDAO {
	return &StockEvtCacheDAO{
		rc: rc,
	}
}

func (self *StockEvtCacheDAO) GetEvts(sym string) ([]*types.CacheStockEvt, error) {
	if strArr, err := self.rc.LRange(self.GetStockEvtCacheKey(sym), 0, -1).Result(); err != nil {
		return nil, err
	} else if len(strArr) == 0 {
		return make([]*types.CacheStockEvt, 0), nil
	} else {
		retArr := make([]*types.CacheStockEvt, 0)
		// str format: priceAction, tradeAt, publishedAt
		for _, str := range strArr {
			tmpArr := strings.Split(str, ",")
			priceAction, _ := strconv.Atoi(tmpArr[0])
			tradeAt, _ := strconv.ParseInt(tmpArr[1], 10, 64)
			var publishedAt int64 = 0
			if len(tmpArr) >= 3 {
				if tmpPublishedAt, err := strconv.ParseInt(tmpArr[2], 10, 64); err != nil {

				} else {
					publishedAt = tmpPublishedAt
				}
			}

			retArr = append(retArr, &types.CacheStockEvt{
				PriceAction: flashcommon.EnumPriceAction(priceAction),
				TradeAt:     tradeAt,
				PublishedAt: publishedAt,
			})
		}
		return retArr, nil
	}
}

func (self *StockEvtCacheDAO) PutEvt(sym string, evt *types.CacheStockEvt) error {
	strVal := fmt.Sprintf("%d,%d,%d", evt.PriceAction, evt.TradeAt, evt.PublishedAt)
	if _, err := self.rc.LPush(self.GetStockEvtCacheKey(sym), strVal).Result(); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *StockEvtCacheDAO) GetStockEvtCacheKey(sym string) string {
	return KeyStockEvtPrefix + sym
}

func (self *StockEvtCacheDAO) ClearAll() error {
	if keys, err := self.rc.Keys(fmt.Sprintf("%s*", KeyStockEvtPrefix)).Result(); err != nil {
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
