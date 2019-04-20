package types

import (
	"sync"
)

type AtomicMap struct {
	data   map[string]string
	rwLock sync.RWMutex
}

func (self *AtomicMap) Get(key string) (string, bool) {
	self.rwLock.RLock()
	defer self.rwLock.RUnlock()
	val, found := self.data[key]
	return val, found
}

func (self *AtomicMap) Set(key, val string) {
	self.rwLock.Lock()
	defer self.rwLock.Unlock()
	self.data[key] = val
}
