package gosync

import "sync"

type SyncMap[K comparable, V any] struct {
	mu     sync.RWMutex
	source map[K]V
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
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

func (sm *SyncMap[K, V]) GetOrDefault(k K, defaultv V) V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.source[k]
	if !ok {
		v = defaultv
	}
	return v
}
func (sm *SyncMap[K, V]) Remove(k K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.source, k)
}
func (sm *SyncMap[K, V]) Clear(k K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.source = make(map[K]V)
}
func (sm *SyncMap[K, V]) Source() map[K]V {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make(map[K]V)
	for k, v := range sm.source {
		result[k] = v
	}
	return result
}

func (sm *SyncMap[K, V]) Values() []V {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make([]V, len(sm.source))
	i := 0
	for _, v := range sm.source {
		result[i] = v
		i++
	}
	return result
}
