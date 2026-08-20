// Package ok provides small test assertions.
//
// Assertions report failures with (testing.TB).Errorf and return whether
// they passed, so a test decides for itself when to stop:
//
//	if !ok.DeepEqual(t, got, want) {
//		return
//	}
//
// The exception is [MustNoError], which calls Fatalf: when a test can't
// get a value it needs, there's rarely a point in continuing.
//
// Most assertions take optional msgAndArgs — a format string followed by
// its arguments — to add context to a failure. [True] is the exception in
// kind: its message replaces the default, because "got false, want true"
// is worth nothing, while everywhere else the message is appended to the
// got/want report. [CmpEqual] is the exception in fact: its variadic slot
// belongs to cmp options.
//
// Equality on comparable types is checked with ==, and assertions that
// pass do not allocate ([DeepEqual], [CmpEqual], and [ErrorAs] excepted;
// they use reflection), whether or not they carry a message. [github.com/google/go-cmp/cmp] and
// [github.com/stefanvanburen/colorcmp] run only after a failure, to
// format the diff.
package ok

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// TB is the subset of [testing.TB] used by assertions.
//
// Fatalf is absent: assertions report through Errorf and never halt the
// test. The one function that halts, [MustNoError], takes a [FatalTB].
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// Equal asserts that got == want.
//
// Note that on pointer types == asserts identity, not value equality: two
// distinct pointers to equal values are not ==. Use [DeepEqual], [CmpEqual],
// or [EqualFunc] to compare what pointers point at.
func Equal[T comparable](tb TB, got, want T, msgAndArgs ...any) bool {
	tb.Helper()
	if got == want {
		return true
	}
	return failPair(tb, got, want, msgAndArgs)
}

// NotEqual asserts that got != want.
func NotEqual[T comparable](tb TB, got, want T, msgAndArgs ...any) bool {
	tb.Helper()
	if got != want {
		return true
	}
	tb.Errorf("got %v, want anything else%s", got, annotate(msgAndArgs))
	return false
}

// DeepEqual asserts that got and want are equal using [reflect.DeepEqual].
// Prefer [Equal] for comparable types: it is faster and stricter.
func DeepEqual[T any](tb TB, got, want T, msgAndArgs ...any) bool {
	tb.Helper()
	if reflect.DeepEqual(got, want) {
		return true
	}
	tb.Errorf("not deeply equal%s:\n%s", annotate(msgAndArgs), diff(tb, got, want))
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
func EqualFunc[T any](tb TB, got, want T, equal func(a, b T) bool, msgAndArgs ...any) bool {
	tb.Helper()
	if equal(got, want) {
		return true
	}
	return failPair(tb, got, want, msgAndArgs)
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
func NoError(tb TB, err error, msgAndArgs ...any) bool {
	tb.Helper()
	if err == nil {
		return true
	}
	tb.Errorf("unexpected error: %v%s", err, annotate(msgAndArgs))
	return false
}

// FatalTB is the subset of [testing.TB] that [MustNoError] requires.
type FatalTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// MustNoError asserts that err is nil, halting the test via Fatalf
// otherwise. Use it to guard values the rest of the test needs:
//
//	u, err := LookupUser("stefan")
//	ok.MustNoError(t, err)
//	ok.Equal(t, u.Name, "stefan")
//
// As with testing.TB's FailNow, MustNoError must run on the test's
// goroutine.
func MustNoError(tb FatalTB, err error, msgAndArgs ...any) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("unexpected error: %v%s", err, annotate(msgAndArgs))
	}
}

// Error asserts that err is non-nil.
func Error(tb TB, err error, msgAndArgs ...any) bool {
	tb.Helper()
	if err != nil {
		return true
	}
	tb.Errorf("got nil, want an error%s", annotate(msgAndArgs))
	return false
}

// ErrorIs asserts that [errors.Is](err, target) is true.
func ErrorIs(tb TB, err, target error, msgAndArgs ...any) bool {
	tb.Helper()
	if errors.Is(err, target) {
		return true
	}
	tb.Errorf("got error %v, want %v in its chain%s", err, target, annotate(msgAndArgs))
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

// ErrorContains asserts that err is non-nil and that its message contains
// substr.
//
// Prefer [ErrorIs] or [ErrorAs] wherever the error offers a sentinel or a
// type to match: message text is rarely part of an API's contract, so a
// test that matches on it breaks when the wording is reworded. This is for
// the errors that give you nothing else to match on.
func ErrorContains(tb TB, err error, substr string, msgAndArgs ...any) bool {
	tb.Helper()
	if err == nil {
		tb.Errorf("got nil, want an error containing %q%s", substr, annotate(msgAndArgs))
		return false
	}
	if msg := err.Error(); !strings.Contains(msg, substr) {
		tb.Errorf("got error %q, want it to contain %q%s", msg, substr, annotate(msgAndArgs))
		return false
	}
	return true
}

// Zero asserts that got is the zero value of its type.
func Zero[T comparable](tb TB, got T, msgAndArgs ...any) bool {
	tb.Helper()
	var zero T
	if got == zero {
		return true
	}
	tb.Errorf("got %v, want zero value%s", got, annotate(msgAndArgs))
	return false
}

// failPair reports the standard got/want failure. It is only called after
// a comparison has failed, so boxing got and want here costs nothing on
// the passing path.
func failPair(tb TB, got, want any, msgAndArgs []any) bool {
	tb.Helper()
	g, w := formatPair(got, want)
	tb.Errorf("got %s, want %s%s", g, w, annotate(msgAndArgs))
	return false
}

// annotate renders the optional msgAndArgs — a format string followed by
// its arguments — as a ": "-prefixed suffix, or "" when there are none.
//
// This differs from [True], where msgAndArgs replace the default message.
// True's "got false, want true" says nothing worth keeping; the assertions
// that use annotate already report got and want, so a caller's context is
// added to that rather than displacing it.
//
// Only ever called after an assertion has failed, so the formatting cost
// never taxes a passing test.
func annotate(msgAndArgs []any) string {
	format, isString := first(msgAndArgs).(string)
	if !isString {
		return ""
	}
	// Copy the args instead of reslicing, for the reason given in True.
	args := make([]any, len(msgAndArgs)-1)
	copy(args, msgAndArgs[1:])
	return ": " + fmt.Sprintf(format, args...)
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
