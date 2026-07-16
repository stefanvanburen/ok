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
)

// TB is the subset of [testing.TB] used by this package.
//
// Fatalf is deliberately absent: assertions never halt the test.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// Equal asserts that got == want.
func Equal[T comparable](tb TB, got, want T) bool {
	tb.Helper()
	if got == want {
		return true
	}
	g, w := formatPair(got, want)
	tb.Errorf("got %s, want %s", g, w)
	return false
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

// EqualFunc asserts that got and want are equal according to equal.
func EqualFunc[T any](tb TB, got, want T, equal func(a, b T) bool) bool {
	tb.Helper()
	if equal(got, want) {
		return true
	}
	g, w := formatPair(got, want)
	tb.Errorf("got %s, want %s", g, w)
	return false
}

// True asserts that got is true.
func True(tb TB, got bool) bool {
	tb.Helper()
	if got {
		return true
	}
	tb.Errorf("got false, want true")
	return false
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
