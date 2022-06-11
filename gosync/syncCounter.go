package gosync

import (
	"sync"
)

type SyncCounter struct {
	couter int64
	mutex  sync.Mutex
}

func GetCounter() *SyncCounter {
	return &SyncCounter{
		couter: 0,
	}
}

func (counter *SyncCounter) Increase() {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()
	counter.couter++
}

func (counter *SyncCounter) Reduce() {
	counter.mutex.Lock()
	defer counter.mutex.Unlock()
	counter.couter--
}
func (counter *SyncCounter) Get() int64 {
	return counter.couter
}
