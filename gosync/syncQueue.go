package gosync

import (
	"container/list"
	"errors"
	"sync"
)

type SyncQueue struct {
	list  *list.List
	mutex sync.Mutex
}

func GetQueue() *SyncQueue {
	return &SyncQueue{
		list: list.New(),
	}
}

func (queue *SyncQueue) Push(data interface{}) {
	if data == nil {
		return
	}
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	queue.list.PushBack(data)
}

func (queue *SyncQueue) Pop() (interface{}, error) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	if element := queue.list.Front(); element != nil {
		queue.list.Remove(element)
		return element.Value, nil
	}
	return nil, errors.New("pop failed")
}

func (queue *SyncQueue) Clear() {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	for element := queue.list.Front(); element != nil; {
		elementNext := element.Next()
		queue.list.Remove(element)
		element = elementNext
	}
}

func (queue *SyncQueue) Len() int {
	return queue.list.Len()
}

// func (queue *SyncQueue) Show() {
// 	for item := queue.list.Front(); item != nil; item = item.Next() {
// 		fmt.Println(item.Value)
// 	}
// }
