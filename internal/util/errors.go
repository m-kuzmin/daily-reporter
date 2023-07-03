package util

import "fmt"

type RecoveredPanicError struct {
	Panic interface{}
}

func (e RecoveredPanicError) Error() string {
	return fmt.Sprintf("panic recovered: %s", e.Panic)
}
