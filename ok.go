// Package ok provides small, non-fatal test assertions.
//
// Every assertion reports failure with (testing.TB).Errorf and returns
// whether it passed, so tests decide for themselves when to stop:
//
//	if !ok.NoError(t, err) {
//		return // can't continue without the value
//	}
//	ok.Equal(t, got.Name, "stefan")
//
// Assertions that pass do not allocate, with the exception of [DeepEqual],
// which uses reflection. Equality on comparable types is checked with ==;
// [github.com/google/go-cmp/cmp] is used only to format diffs after a
// failure.
package ok

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
)

// TB is the subset of [testing.TB] used by this package.
//
// Fatalf is deliberately absent: assertions never halt the test.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// Equal asserts that got == want.
//
// Note that on pointer types == asserts identity, not value equality: two
// distinct pointers to equal values are not ==. Use [DeepEqual], [CmpEqual],
// or [EqualFunc] to compare what pointers point at.
func Equal[T comparable](tb TB, got, want T) bool {
	tb.Helper()
	if got == want {
		return true
	}
	return failPair(tb, got, want)
}

// NotEqual asserts that got != want.
func NotEqual[T comparable](tb TB, got, want T) bool {
	tb.Helper()
	if got != want {
		return true
	}
	tb.Errorf("got %v, want anything else", got)
	return false
}

// DeepEqual asserts that got and want are equal using [reflect.DeepEqual].
// Prefer [Equal] for comparable types: it is faster and stricter.
func DeepEqual[T any](tb TB, got, want T) bool {
	tb.Helper()
	if reflect.DeepEqual(got, want) {
		return true
	}
	tb.Errorf("not deeply equal:\n%s", diff(tb, got, want))
	return false
}

// CmpEqual asserts that got and want are equal using
// [github.com/google/go-cmp/cmp.Equal] with opts, e.g. protocmp.Transform
// for protobuf messages. Unlike the other assertions, it pays cmp's
// reflection cost even when the assertion passes.
//
// cmp panics when opts don't cover a type it can't otherwise compare (e.g.
// a struct with unexported fields); CmpEqual lets that panic propagate, as
// cmp's message names the missing option.
func CmpEqual[T any](tb TB, got, want T, opts ...cmp.Option) bool {
	tb.Helper()
	if cmp.Equal(got, want, opts...) {
		return true
	}
	tb.Errorf("not equal:\n%s", diff(tb, got, want, opts...))
	return false
}

// EqualFunc asserts that got and want are equal according to equal.
func EqualFunc[T any](tb TB, got, want T, equal func(a, b T) bool) bool {
	tb.Helper()
	if equal(got, want) {
		return true
	}
	return failPair(tb, got, want)
}

// True asserts that got is true. The optional msgAndArgs — a format string
// followed by its arguments — replace the default failure message, letting
// predicates report runtime values:
//
//	ok.True(t, got > limit, "got %d, want > %d", got, limit)
func True(tb TB, got bool, msgAndArgs ...any) bool {
	tb.Helper()
	if got {
		return true
	}
	if format, isString := first(msgAndArgs).(string); isString {
		// Copy the args instead of reslicing: passing msgAndArgs itself to
		// Errorf makes the parameter escape, which would heap-allocate the
		// caller's variadic slice even when the assertion passes.
		args := make([]any, len(msgAndArgs)-1)
		copy(args, msgAndArgs[1:])
		tb.Errorf(format, args...)
	} else {
		tb.Errorf("got false, want true")
	}
	return false
}

func first(s []any) any {
	if len(s) == 0 {
		return nil
	}
	return s[0]
}

// Panics asserts that f panics, returning the recovered value. Assert on
// the value for testify's PanicsWithValue:
//
//	v, _ := ok.Panics(t, func() { mustParse("bogus") })
//	ok.Equal(t, v, any("bogus input"))
func Panics(tb TB, f func()) (recovered any, panicked bool) {
	tb.Helper()
	returned := false
	func() {
		defer func() { recovered = recover() }()
		f()
		returned = true
	}()
	if returned {
		tb.Errorf("function did not panic")
	}
	return recovered, !returned
}

// NoError asserts that err is nil.
func NoError(tb TB, err error) bool {
	tb.Helper()
	if err == nil {
		return true
	}
	tb.Errorf("unexpected error: %v", err)
	return false
}

// FatalTB is the subset of [testing.TB] that [MustNoError] requires.
type FatalTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// MustNoError asserts that err is nil, halting the test via Fatalf
// otherwise. It is the guard for errors the rest of the test cannot
// proceed past — fatality in this package exists only here, on the error
// path, where the control-flow dependency actually lives:
//
//	u, err := LookupUser("stefan")
//	ok.MustNoError(t, err)
//	ok.Equal(t, u.Name, "stefan")
//
// As with testing.TB's FailNow, MustNoError must run on the test's
// goroutine.
func MustNoError(tb FatalTB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("unexpected error: %v", err)
	}
}

// Error asserts that err is non-nil.
func Error(tb TB, err error) bool {
	tb.Helper()
	if err != nil {
		return true
	}
	tb.Errorf("got nil, want an error")
	return false
}

// ErrorIs asserts that [errors.Is](err, target) is true.
func ErrorIs(tb TB, err, target error) bool {
	tb.Helper()
	if errors.Is(err, target) {
		return true
	}
	tb.Errorf("got error %v, want %v in its chain", err, target)
	return false
}

// ErrorAs asserts that err's chain contains an error of type T, returning
// that error if found.
func ErrorAs[T error](tb TB, err error) (T, bool) {
	tb.Helper()
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	tb.Errorf("got error %v, want %T in its chain", err, target)
	return target, false
}

// Zero asserts that got is the zero value of its type.
func Zero[T comparable](tb TB, got T) bool {
	tb.Helper()
	var zero T
	if got == zero {
		return true
	}
	tb.Errorf("got %v, want zero value", got)
	return false
}

// failPair reports the standard got/want failure. It is only called after
// a comparison has failed, so boxing got and want here costs nothing on
// the passing path.
func failPair(tb TB, got, want any) bool {
	tb.Helper()
	g, w := formatPair(got, want)
	tb.Errorf("got %s, want %s", g, w)
	return false
}

// formatPair renders two unequal values for a failure message. When their
// %v forms are indistinguishable (e.g. a string vs. a fmt.Stringer that
// prints the same), it falls back to %#v so the difference is visible.
func formatPair(got, want any) (string, string) {
	g, w := fmt.Sprintf("%v", got), fmt.Sprintf("%v", want)
	if g == w {
		g, w = fmt.Sprintf("%#v", got), fmt.Sprintf("%#v", want)
	}
	return g, w
}
