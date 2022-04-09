package syncmap

import "sync"

type syncMap[K comparable, V comparable] struct {
	mu     sync.RWMutex
	source map[K]V
}

func NewSyncMap[K comparable, V comparable]() *syncMap[K, V] {
	result := &syncMap[K, V]{
		source: make(map[K]V),
	}
	return result
}

func (sm *syncMap[K, V]) Put(k K, v V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.source[k] = v
}
func (sm *syncMap[K, V]) Get(k K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.source[k]
	return v, ok
}
func (sm *syncMap[K, V]) Remove(k K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.source, k)
}
