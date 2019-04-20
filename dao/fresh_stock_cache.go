package dao

import (
	// "gitlab.wallstcn.com/baoer/flash/flashstd"
	// "gitlab.wallstcn.com/baoer/flash/flashtick/enums"
	"gopkg.in/redis.v5"
	// "strings"
)

type FreshStockCacheDAO struct {
	rc *redis.Client
}

func NewFreshStockCacheDAO(rc *redis.Client) *FreshStockCacheDAO {
	return &FreshStockCacheDAO{
		rc: rc,
	}
}

func (self *FreshStockCacheDAO) Add(sym string, dateStr string) error {
	if _, err := self.rc.HSet(KeyFreshStockHash, sym, dateStr).Result(); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *FreshStockCacheDAO) GetAllAsMap() (map[string]string, error) {
	if symToDateStrMap, err := self.rc.HGetAll(KeyFreshStockHash).Result(); err != nil {
		return nil, err
	} else {
		return symToDateStrMap, nil
	}
}

func (self *FreshStockCacheDAO) MarkForDeletion(sym string) error {
	if _, err := self.rc.HSet(KeyMarkedFreshStockHash, sym, "del").Result(); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *FreshStockCacheDAO) RemMarkedStocks() error {
	if symMap, err := self.rc.HGetAll(KeyMarkedFreshStockHash).Result(); err != nil {
		return err
	} else {
		allSyms := []string{}
		for sym, _ := range symMap {
			allSyms = append(allSyms, sym)
		}
		if len(allSyms) == 0 {
			return nil
		}
		if _, err := self.rc.HDel(KeyFreshStockHash, allSyms...).Result(); err != nil {
			return err
		} else { // delete marked fresh stock hash itself
			if _, err := self.rc.Del(KeyMarkedFreshStockHash).Result(); err != nil {
				return err
			}
		}
	}
	return nil
}
