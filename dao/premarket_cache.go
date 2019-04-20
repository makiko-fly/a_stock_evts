package dao

import (
	// "gitlab.wallstcn.com/baoer/flash/flashcommon"
	"errors"
	"fmt"

	"gitlab.wallstcn.com/baoer/flash/flashstd/json"
	"gitlab.wallstcn.com/baoer/flash/flashtick/types"
	"gopkg.in/redis.v5"
	// "strings"
)

type PremarketCacheDAO struct {
	rc *redis.Client
}

func NewPremarketCacheDAO(rc *redis.Client) *PremarketCacheDAO {
	return &PremarketCacheDAO{
		rc: rc,
	}
}

func (self *PremarketCacheDAO) GetBySym(sym string) (*types.PremarketStat, error) {
	if strVal, err := self.rc.HGet(KeyPremarketStatHash, sym).Result(); err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	} else if strVal == "" {
		return nil, errors.New(fmt.Sprintf("Value for hash %s, hashkey: %s is empty string", KeyPremarketStatHash, sym))
	} else {
		var premarketStat types.PremarketStat
		if err := json.Unmarshal([]byte(strVal), &premarketStat); err != nil {
			return nil, err
		} else {
			return &premarketStat, nil
		}
	}
}

func (self *PremarketCacheDAO) UpdateStat(sym string, premarketStat *types.PremarketStat) error {
	if byteArr, err := json.Marshal(premarketStat); err != nil {
		return err
	} else {
		if _, err := self.rc.HSet(KeyPremarketStatHash, sym, string(byteArr)).Result(); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (self *PremarketCacheDAO) ClearAll() error {
	if _, err := self.rc.Del(KeyPremarketStatHash).Result(); err != nil {
		return err
	} else {
		return nil
	}
}

// ====================================================================================
// this cache maintains another info: TradeStartEvtSent
func (self *PremarketCacheDAO) GetTradeStartEvtSent() (bool, error) {
	if strVal, err := self.rc.Get(KeyTradeStartEvtSent).Result(); err != nil {
		if err == redis.Nil { // key not found
			return false, nil
		} else {
			return false, err
		}
	} else if strVal == "1" {
		return true, nil
	} else {
		return false, nil
	}
}

func (self *PremarketCacheDAO) SetTradeStartEvtSent(sent bool) error {
	strVal := "1"
	if !sent {
		strVal = "0"
	}
	if _, err := self.rc.Set(KeyTradeStartEvtSent, strVal, -1).Result(); err != nil {
		return err
	} else {
		return nil
	}
}
