package option

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// Option is a type can be in one of two states: `Some(T)` or `None`.
//
// https://doc.rust-lang.org/std/option/
type Option[T any] struct {
	value interface{} //nolint:structcheck // Says value is unused, when it is.
}

// Some(T) creates an Option that holds a value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value}
}

// None[T]() creates a *typed* Option that doesnt hold anything.
func None[T any]() Option[T] {
	return Option[T]{}
}

// IsSome returns true if the Option is a `Some(T)` variant.
func (o Option[T]) IsSome() bool {
	return o.value != nil
}

// IsNone returns true if the Option is a `None` variant.
func (o Option[T]) IsNone() bool {
	return o.value == nil
}

// If the Option is `Some` returns `Some(f(T))`. Otherwise `None`.
func (o Option[T]) Map(f func(T) T) Option[T] {
	if v, isSome := o.Unwrap(); isSome {
		o.value = f(v)
	}

	return o
}

// Returns `Some(f(T))` if the Option is `Some`. Otherwise `None`.
func Map[T, U any](o Option[T], f func(T) U) Option[U] {
	if v, isSome := o.Unwrap(); isSome {
		return Some(f(v))
	}

	return None[U]()
}

// Unwrap returns the value contained in `Some` and true; or a zero value and false if the Option is None.
func (o Option[T]) Unwrap() (T, bool) {
	if o.IsSome() {
		return o.value.(T), true //nolint:forcetypeassert // Type T is guaranteed
	}

	var none T

	return none, false
}

// If the Option is `Some` returns contained value. Otherwise returns `ifNone`. If the value is a result of a function
// call use UnwrapOrElse for lazy evaluation.
func (o Option[T]) UnwrapOr(ifNone T) T {
	if v, isSome := o.Unwrap(); isSome {
		return v
	}

	return ifNone
}

/*
If the Option is `Some` returns the contained value. If None executes f and returns the result.

Note that the arguments to functions inside f are not lazily evaluated.
*/
func (o Option[T]) UnwrapOrElse(f func() T) T {
	if v, isSome := o.Unwrap(); isSome {
		return v
	}

	return f()
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

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if t, isSome := o.Unwrap(); isSome {
		marshaled, err := json.Marshal(t)

		return marshaled, errors.Wrapf(err, "while marshaling %#v", t)
	}

	return []byte("null"), nil
}
