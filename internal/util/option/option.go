package option

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// https://doc.rust-lang.org/std/option/
type Option[T any] struct {
	value interface{} //nolint:structcheck // Says value is unused, when it is.
}

func Some[T any](value T) Option[T] {
	return Option[T]{value: value}
}

func None[T any]() Option[T] {
	return Option[T]{}
}

func (o Option[T]) IsSome() bool {
	return o.value != nil
}

func (o Option[T]) IsNone() bool {
	return o.value == nil
}

func (o Option[T]) Unwrap() (T, bool) {
	if o.IsSome() {
		return o.value.(T), true //nolint:forcetypeassert // Type T is guaranteed
	}

	var none T

	return none, false
}

func (o Option[T]) UnwrapOr(ifNone T) T {
	if v, isSome := o.Unwrap(); isSome {
		return v
	}

	return ifNone
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	var parsed T

	err := json.Unmarshal(data, &parsed)
	if err != nil {
		return fmt.Errorf("while JSON-unmarshaling Option[%T]: %w", parsed, err)
	}

	*o = Some[T](parsed)

	return nil
}

func (o *Option[T]) MarshalJSON() ([]byte, error) {
	if t, isSome := o.Unwrap(); isSome {
		marshaled, err := json.Marshal(t)

		return marshaled, errors.Wrapf(err, "while marshaling %#v", t)
	}

	return make([]byte, 0), nil
}
