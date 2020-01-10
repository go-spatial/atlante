package setqueue

import (
	"sync"

	"github.com/gdey/errors"
)

const (
	// ErrDuplicate is returned when the key already exists in the queue
	ErrDuplicate = errors.String("key in queue")
	// ErrNoSpace is returned when the queue is full
	ErrNoSpace = errors.String("queue is full")
	// ErrEmptyQueue is returned when an element is requested but the queue is empty
	ErrEmptyQueue = errors.String("queue is empty")
	// ErrNotFound is returned when the requested key is not in the queue.
	ErrNotFound = errors.String("element not found")
)

type Value interface{}

type kv struct {
	key string
	val Value
}

// Q is a queue
type Q struct {
	lck  sync.RWMutex
	ueue []*kv
}

// New Creates a new queue with the given size
func New(size int) Q {
	return Q{
		ueue: make([]*kv, 0, size),
	}
}

// lookup will find the index of the  element in the queue.
// Note, locking it left as the responsiblity of the caller.
func (q *Q) lookup(key string) int {
	if q == nil || len(q.ueue) == 0 {
		return -1
	}
	for i, kv := range q.ueue {
		if kv == nil {
			continue
		}
		if kv.key != key {
			continue
		}
		return i
	}
	return -1
}

// Get will return the item store in the queue and true, or the nil and false
func (q *Q) Get(key string) (interface{}, bool) {
	q.lck.RLock()
	defer q.lck.RUnlock()
	idx := q.lookup(key)
	if idx == -1 {
		return nil, false
	}
	return q.ueue[idx].val, true
}

// Push will push the given element to the end of the queue, if it isn't already in the queue.
// if the key is in the queue, we will return an ErrDuplicate
// if the queue is full ErrNoSpace will be returned
func (q *Q) Push(key string, val Value) error {
	q.lck.Lock()
	defer q.lck.Unlock()
	if idx := q.lookup(key); idx != -1 {
		return ErrDuplicate
	}
	if cap(q.ueue) == len(q.ueue) {
		return ErrNoSpace
	}
	q.ueue = append(q.ueue, &kv{
		key: key,
		val: val,
	})
	return nil
}

// Pop will remove and return the first item in the queue if there is one.
// If there isn't an item it will return an error of ErrEmptyQueue
func (q *Q) Pop() (key string, val Value, err error) {
	q.lck.Lock()
	defer q.lck.Unlock()
	if len(q.ueue) == 0 {
		return "", nil, ErrEmptyQueue
	}
	kv := q.remove(0)
	return kv.key, kv.val, nil
}

func (q *Q) remove(idx int) (kv *kv) {
	kv = q.ueue[idx]
	copy(q.ueue[idx:], q.ueue[idx+1:])
	q.ueue = q.ueue[0 : len(q.ueue)-1]
	return kv
}

// Remove will remove the element with the key and return the value, or nil if the value
// does not exists.
func (q *Q) Remove(key string) (val Value, found bool) {
	q.lck.Lock()
	defer q.lck.Unlock()
	idx := q.lookup(key)
	if idx == -1 {
		return nil, false
	}
	kv := q.remove(idx)
	return kv.val, true
}
