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
// Every assertion but [CmpEqual], whose variadic slot belongs to cmp
// options, takes [Option] values that add context to a failure. [Sprintf]
// is the one that exists:
//
//	ok.Equal(t, len(matches), 2, ok.Sprintf("matches for %q", query))
//	got 3, want 2: matches for "Foo"
//
// An option is a type rather than a trailing format string and args so
// that go vet can check the format, which it cannot do when the format is
// buried in a ...any. See [Sprintf].
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
func Equal[T comparable](tb TB, got, want T, opts ...Option) bool {
	tb.Helper()
	if got == want {
		return true
	}
	return failPair(tb, got, want, opts)
}

// NotEqual asserts that got != want.
func NotEqual[T comparable](tb TB, got, want T, opts ...Option) bool {
	tb.Helper()
	if got != want {
		return true
	}
	tb.Errorf("got %v, want anything else%s", got, annotate(opts))
	return false
}

// DeepEqual asserts that got and want are equal using [reflect.DeepEqual].
// Prefer [Equal] for comparable types: it is faster and stricter.
func DeepEqual[T any](tb TB, got, want T, opts ...Option) bool {
	tb.Helper()
	if reflect.DeepEqual(got, want) {
		return true
	}
	tb.Errorf("not deeply equal%s:\n%s", annotate(opts), diff(tb, got, want))
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
func EqualFunc[T any](tb TB, got, want T, equal func(a, b T) bool, opts ...Option) bool {
	tb.Helper()
	if equal(got, want) {
		return true
	}
	return failPair(tb, got, want, opts)
}

// True asserts that got is true.
//
// The default failure message says only "got false, want true", so an
// [Sprintf] option is usually worth adding to name what was expected:
//
//	ok.True(t, got > limit, ok.Sprintf("got %d, want > %d", got, limit))
func True(tb TB, got bool, opts ...Option) bool {
	tb.Helper()
	if got {
		return true
	}
	tb.Errorf("got false, want true%s", annotate(opts))
	return false
}

// Panics asserts that f panics, returning the recovered value. Assert on
// the value for testify's PanicsWithValue:
//
//	v, _ := ok.Panics(t, func() { mustParse("bogus") })
//	ok.Equal(t, v, any("bogus input"))
func Panics(tb TB, f func(), opts ...Option) (recovered any, panicked bool) {
	tb.Helper()
	returned := false
	func() {
		defer func() { recovered = recover() }()
		f()
		returned = true
	}()
	if returned {
		tb.Errorf("function did not panic%s", annotate(opts))
	}
	return recovered, !returned
}

// NoError asserts that err is nil.
func NoError(tb TB, err error, opts ...Option) bool {
	tb.Helper()
	if err == nil {
		return true
	}
	tb.Errorf("unexpected error: %v%s", err, annotate(opts))
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
func MustNoError(tb FatalTB, err error, opts ...Option) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("unexpected error: %v%s", err, annotate(opts))
	}
}

// Error asserts that err is non-nil.
func Error(tb TB, err error, opts ...Option) bool {
	tb.Helper()
	if err != nil {
		return true
	}
	tb.Errorf("got nil, want an error%s", annotate(opts))
	return false
}

// ErrorIs asserts that [errors.Is](err, target) is true.
func ErrorIs(tb TB, err, target error, opts ...Option) bool {
	tb.Helper()
	if errors.Is(err, target) {
		return true
	}
	tb.Errorf("got error %v, want %v in its chain%s", err, target, annotate(opts))
	return false
}

// ErrorAs asserts that err's chain contains an error of type T, returning
// that error if found.
func ErrorAs[T error](tb TB, err error, opts ...Option) (T, bool) {
	tb.Helper()
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	tb.Errorf("got error %v, want %T in its chain%s", err, target, annotate(opts))
	return target, false
}

// ErrorContains asserts that err is non-nil and that its message contains
// substr.
//
// Prefer [ErrorIs] or [ErrorAs] wherever the error offers a sentinel or a
// type to match: message text is rarely part of an API's contract, so a
// test that matches on it breaks when the wording is reworded. This is for
// the errors that give you nothing else to match on.
func ErrorContains(tb TB, err error, substr string, opts ...Option) bool {
	tb.Helper()
	if err == nil {
		tb.Errorf("got nil, want an error containing %q%s", substr, annotate(opts))
		return false
	}
	if msg := err.Error(); !strings.Contains(msg, substr) {
		tb.Errorf("got error %q, want it to contain %q%s", msg, substr, annotate(opts))
		return false
	}
	return true
}

// Zero asserts that got is the zero value of its type.
func Zero[T comparable](tb TB, got T, opts ...Option) bool {
	tb.Helper()
	var zero T
	if got == zero {
		return true
	}
	tb.Errorf("got %v, want zero value%s", got, annotate(opts))
	return false
}

// failPair reports the standard got/want failure. It is only called after
// a comparison has failed, so boxing got and want here costs nothing on
// the passing path.
func failPair(tb TB, got, want any, opts []Option) bool {
	tb.Helper()
	g, w := formatPair(got, want)
	tb.Errorf("got %s, want %s%s", g, w, annotate(opts))
	return false
}

// Option adds context to an assertion's failure message. [Sprintf] is the
// only one today; the type exists so that adding another later does not
// change every signature.
type Option struct {
	format string
	args   []any
}

// Sprintf returns an [Option] that appends a formatted message to a
// failure, so an assertion can name the case it was checking:
//
//	ok.Equal(t, len(matches), 2, ok.Sprintf("matches for %q", query))
//	got 3, want 2: matches for "Foo"
//
// The arguments are held, not formatted, until an assertion actually
// fails, so a passing assertion still does not allocate.
//
// Because the format string sits in a fixed position here rather than
// inside a ...any, go vet's printf analysis can check it:
//
//	go vet -printf.funcs=Sprintf ./...
func Sprintf(format string, args ...any) Option {
	return Option{format: format, args: args}
}

// annotate renders opts as a ": "-prefixed suffix, or "" when there are
// none. Only ever called after an assertion has failed, so the formatting
// cost never taxes a passing test.
func annotate(opts []Option) string {
	for _, o := range opts {
		if o.format == "" {
			continue
		}
		// Copy the args instead of passing o.args straight to Sprintf:
		// letting them escape here would heap-allocate the caller's
		// variadic slice even when the assertion passes.
		args := make([]any, len(o.args))
		copy(args, o.args)
		return ": " + fmt.Sprintf(o.format, args...)
	}
	return ""
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
