/*
borrowonce provides a storage for lending values with a promise to return them. If the value was not returned no one
else can borrow it. Here is an ELI5:

Suppose you have a box that has a ball inside. You then have people (threads) that want to play with the ball. The first
person takes the ball. Then someone else looks in the box and there is no ball inside. They wait for the box to have the
ball again. Then someone else comes in and also wants the ball. But they came later so they will wait until the ball is
given to the person before and then wait until that person is done playing with the ball.
*/
package borrowonce

import (
	"fmt"
	"sync"
)

/*
Storage lends values in such a way that the first borrower will immediately get the value, and all borrowers after will
get the value in the same order they requested it, but only after the previous borrower has returned the value.

This is useful if you generally dont operate on the same key in the map, but if you have to do so threads can wait for
the previous thread to be finished mutating the value and be sure that their operation on the value is in the correct
order.
*/
type Storage[K comparable, V any] struct {
	storeMu *sync.Mutex         //nolint:structcheck // Is used!
	store   map[K]borrowable[V] //nolint:structcheck // Is used!
}

// NewStorage creates a new empty storage
func NewStorage[K comparable, V any]() Storage[K, V] {
	return Storage[K, V]{
		store: make(map[K]borrowable[V]),
	}
}

// Set stores the value in the map. Panics if it already exists. Use Return for keys that are in the map already.
func (s *Storage[K, V]) Set(key K, value V) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if _, exists := s.store[key]; exists {
		panic(fmt.Sprintf("Tried to set an existing value in borrowonce.Storage[%T, %T]. Use Release instead.",
			*new(K), *new(V),
		))
	}

	s.store[key] = borrowable[V]{
		value:    value,
		queue:    make([]*Future[V], 0),
		borrowed: false,
	}
}

/*
Borrow gives you a Future which is like a position in the queue to access and mutate the value. If the key doesnt exist
in the map you will get `nil, false`.
*/
func (s *Storage[K, V]) Borrow(key K) (*Future[V], bool) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	value, found := s.store[key]
	if !found {
		return nil, false
	}

	future := &Future[V]{
		vMu: sync.Mutex{},
		v:   *new(V),
	}

	if len(value.queue) == 0 && !value.borrowed {
		value.borrowed = true
		s.store[key] = value
		future.v = value.value

		return future, found
	}

	future.vMu.Lock()
	value.queue = append(value.queue, future)
	s.store[key] = value

	return future, true
}

// Return stores the value in the map and allows the next borrower to have it.
func (s *Storage[K, V]) Return(key K, value V) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	lockable, exists := s.store[key]
	if !exists {
		panic(fmt.Sprintf("Tried to release a key that isn't in borrowonce.Storage[%T, %T]. Use Set instead.",
			*new(K), *new(V)))
	}

	lockable.value = value
	if len(lockable.queue) == 0 {
		lockable.borrowed = false
	} else {
		lockable.queue[0].v = value
		lockable.queue[0].vMu.Unlock()
		lockable.queue = lockable.queue[1:]
	}

	s.store[key] = lockable
}

/*
Future allows you request a position in the borrow queue and Wait() your turn.
*/
type Future[V any] struct {
	vMu sync.Mutex //nolint:structcheck // Is used!
	v   V          //nolint:structcheck // Is used!
}

func NewImmediateFuture[V any](v V) *Future[V] {
	return &Future[V]{v: v}
}

/*
Wait will return the value once it is your turn to have it. After you are done with it you have to call Storage.Return
*/
func (f *Future[V]) Wait() V { //nolint:golint // Is confusing Storage and Future
	f.vMu.Lock()
	defer f.vMu.Unlock()

	return f.v
}

/*
borrowable stores the current version of the value as well as a list of borrowers. Once the value is returned it will be
updated and the next borrower will get that new version.
*/
type borrowable[V any] struct {
	value    V            //nolint:structcheck // Current value
	borrowed bool         //nolint:structcheck // If there is 1 borrower
	queue    []*Future[V] //nolint:structcheck // List of borrowers
}
