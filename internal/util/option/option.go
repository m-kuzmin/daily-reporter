package option

import (
	"encoding/json"
	"fmt"
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

func (o Option[T]) MustUnwrap() T {
	if o.IsSome() {
		return o.value.(T) //nolint:forcetypeassert // Type T is guaranteed
	}

	var t T

	panic(fmt.Sprintf("MustUnwrap called on a None Option of type %T.", t))
}

func (o Option[T]) Unwrap() (T, bool) {
	if o.IsSome() {
		return o.value.(T), true //nolint:forcetypeassert // Type T is guaranteed
	}

	var none T

	return none, false
}

func (o Option[T]) UnwrapOr(ifNone T) T {
	if o.IsSome() {
		return o.value.(T) //nolint:forcetypeassert // Type T is guaranteed
	}

	return ifNone
}

func Flatmap[T, U any](o Option[T], f func(T) U) Option[U] {
	if v, isSome := o.Unwrap(); isSome {
		return Some(f(v))
	}

	return None[U]()
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
