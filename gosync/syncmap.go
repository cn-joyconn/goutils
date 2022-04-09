package gosync

import "sync"

type SyncMap[K comparable, V comparable] struct {
	mu     sync.RWMutex
	source map[K]V
}

func NewSyncMap[K comparable, V comparable]() *SyncMap[K, V] {
	result := &SyncMap[K, V]{
		source: make(map[K]V),
	}
	return result
}

func (sm *SyncMap[K, V]) Put(k K, v V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.source[k] = v
}
func (sm *SyncMap[K, V]) Get(k K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.source[k]
	return v, ok
}
func (sm *SyncMap[K, V]) Remove(k K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.source, k)
}
