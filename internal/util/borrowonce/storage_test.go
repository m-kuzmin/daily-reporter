package borrowonce_test

import (
	"testing"
	"time"

	"github.com/m-kuzmin/daily-reporter/internal/util/borrowonce"
)

const (
	key   = "key"
	value = "value"
)

func TestGetEmpty(t *testing.T) {
	t.Parallel()

	store := borrowonce.NewStorage[string, string]()

	future, exists := store.Borrow(key)
	if exists {
		t.Logf("%q", future.Wait())
		t.Fail()
	}

	future, prevGetCreatedValue := store.Borrow(value)
	if prevGetCreatedValue {
		t.Logf("%q", future.Wait())
		t.Fail()
	}
}

func TestSet(t *testing.T) {
	t.Parallel()

	store := borrowonce.NewStorage[string, string]()

	store.Set(key, value)

	future, found := store.Borrow(key)

	if v := future.Wait(); !found || v != value {
		t.Logf("%q", v)
		t.Fail()
	}
}

func TestGetLeasedOnceWouldBlock(t *testing.T) {
	t.Parallel()

	store := borrowonce.NewStorage[string, string]()

	store.Set(key, value)
	store.Borrow(key) // 1

	finished := false
	future, _ := store.Borrow(key) // 2

	go func() {
		t.Log("Before blocking")
		future.Wait() // 1
		t.Log("After blocking")

		finished = true
	}()

	time.Sleep(300 * time.Millisecond) // should be enough to know that Get didn't return yet

	if finished {
		t.Fatal("Wait should have blocked.")
	}

	store.Return(key, value) // 1

	time.Sleep(300 * time.Millisecond)

	if !finished {
		t.Fatal("Wait should've unblocked by now.")
	}

	store.Return(key, value) // 2
}

func TestGetLeasedTwiceWouldBlock(t *testing.T) {
	t.Parallel()

	store := borrowonce.NewStorage[string, string]()

	store.Set(key, value)
	store.Borrow(key) // 1

	finished1 := false
	finished2 := false

	future1, _ := store.Borrow(key) // 2
	future2, _ := store.Borrow(key) // 3

	go func() {
		t.Log("1 Before blocking")
		future1.Wait() // 1
		t.Log("1 After blocking")

		finished1 = true
	}()
	go func() {
		t.Log("2 Before blocking")
		future2.Wait() // 2
		t.Log("2 After blocking")

		finished2 = true
	}()

	time.Sleep(300 * time.Millisecond) // should be enough to know that Get didn't return yet

	if finished1 || finished2 {
		t.Fatalf("Wait should have blocked. 1: %t 2: %t", finished1, finished2)
	}

	store.Return(key, value) // 1

	time.Sleep(300 * time.Millisecond)

	if !finished1 {
		t.Fatal("1 Should've unblocked by now.")
	}

	store.Return(key, value) // 2

	time.Sleep(300 * time.Millisecond)

	if !finished2 {
		t.Fatal("2 Should've unblocked by now.")
	}

	store.Return(key, value) // 3
}

func TestFutureAwaitReturnsUpdatedValue(t *testing.T) {
	t.Parallel()

	store := borrowonce.NewStorage[string, string]()

	store.Set(key, "original")
	future, _ := store.Borrow(key)
	nextFuture, _ := store.Borrow(key)

	future.Wait()
	store.Return(key, "new")

	if latestValue := nextFuture.Wait(); latestValue != "new" {
		t.Fatalf("The value has not been updated, it's: %q", latestValue)
	}
}
