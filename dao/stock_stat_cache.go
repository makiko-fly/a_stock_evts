package dao

import (
	"errors"
	"fmt"

	"gitlab.wallstcn.com/baoer/flash/flashstd/json"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
	"gopkg.in/redis.v5"
	// "strings"
)

type StockStatCacheDAO struct {
	rc *redis.Client
}

func NewStockStatCacheDAO(rc *redis.Client) *StockStatCacheDAO {
	return &StockStatCacheDAO{
		rc: rc,
	}
}

func (self *StockStatCacheDAO) GetBySym(sym string) (*types.StockStat, error) {
	if strVal, err := self.rc.HGet(KeyStockStatHash, sym).Result(); err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	} else if strVal == "" {
		return nil, errors.New(fmt.Sprintf("Value for hash %s, hashkey: %s is empty string", KeyStockStatHash, sym))
	} else {
		var stockStat types.StockStat
		if err := json.Unmarshal([]byte(strVal), &stockStat); err != nil {
			return nil, err
		} else {
			return &stockStat, nil
		}
	}
}

func (self *StockStatCacheDAO) UpdateStat(sym string, stockStat *types.StockStat) error {
	if byteArr, err := json.Marshal(stockStat); err != nil {
		return err
	} else {
		if _, err := self.rc.HSet(KeyStockStatHash, sym, string(byteArr)).Result(); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (self *StockStatCacheDAO) ClearAll() error {
	if _, err := self.rc.Del(KeyStockStatHash).Result(); err != nil {
		return err
	} else {
		return nil
	}
}
