package ok

import (
	"cmp"
	"reflect"
	"slices"
)

// Must returns a [TB] that halts the test at the first assertion that fails
// through it: its Errorf reports through tb's Fatalf. Assertions passed tb
// itself keep going, so a test chooses where to stop by which one it hands
// an assertion:
//
//	must := ok.Must(t)
//	ok.Equal(must, len(items), 3) // halts here on failure
//	ok.Equal(t, items[0], "a")    // reports and carries on
//
// Declare it once and reuse it: converting the wrapper to a [TB] allocates,
// so ok.Must(t) written inline at each assertion would cost an allocation
// per call. [MustNoError] is ok.NoError(ok.Must(tb), err) in one call.
func Must(tb FatalTB) TB { return must{tb} }

// must embeds its FatalTB so that Helper is the embedded method itself: a
// Helper of its own would mark that forwarding method as the helper, and
// failures would be attributed to a line inside this package instead of the
// test's.
type must struct{ FatalTB }

func (m must) Errorf(format string, args ...any) {
	m.Helper()
	m.Fatalf(format, args...)
}

// Nil asserts that got is nil: an untyped nil, or a nil pointer, slice,
// map, channel, function or interface — including an interface holding a
// nil pointer, which == nil does not count as nil.
//
// got passes as an any, and converting a value that is not pointer-shaped
// to one allocates; the check itself uses reflection.
func Nil(tb TB, got any, opts ...Option) bool {
	tb.Helper()
	if isNil(got) {
		return true
	}
	tb.Errorf("got %v, want nil%s", got, annotate(opts))
	return false
}

// NotNil asserts that got is not nil, by [Nil]'s definition.
func NotNil(tb TB, got any, opts ...Option) bool {
	tb.Helper()
	if !isNil(got) {
		return true
	}
	tb.Errorf("got nil, want non-nil%s", annotate(opts))
	return false
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

// Len asserts that s has n elements, and shows them when it does not:
//
//	got 3 elements [a b c], want 2
func Len[S ~[]E, E any](tb TB, s S, n int, opts ...Option) bool {
	tb.Helper()
	if len(s) == n {
		return true
	}
	// Formatting a copy keeps s itself from escaping, which would otherwise
	// heap-allocate the caller's slice even when Len passes.
	elems := slices.Clone(s)
	tb.Errorf("got %d elements %v, want %d%s", len(s), elems, n, annotate(opts))
	return false
}

// Greater asserts that got > bound.
func Greater[T cmp.Ordered](tb TB, got, bound T, opts ...Option) bool {
	tb.Helper()
	if got > bound {
		return true
	}
	return failOrder(tb, got, ">", bound, opts)
}

// GreaterOrEqual asserts that got >= bound.
func GreaterOrEqual[T cmp.Ordered](tb TB, got, bound T, opts ...Option) bool {
	tb.Helper()
	if got >= bound {
		return true
	}
	return failOrder(tb, got, ">=", bound, opts)
}

// Less asserts that got < bound.
func Less[T cmp.Ordered](tb TB, got, bound T, opts ...Option) bool {
	tb.Helper()
	if got < bound {
		return true
	}
	return failOrder(tb, got, "<", bound, opts)
}

// LessOrEqual asserts that got <= bound.
func LessOrEqual[T cmp.Ordered](tb TB, got, bound T, opts ...Option) bool {
	tb.Helper()
	if got <= bound {
		return true
	}
	return failOrder(tb, got, "<=", bound, opts)
}

// failOrder reports a failed ordering. Like failPair, it is only reached
// after the comparison failed, so boxing the operands costs a passing
// assertion nothing.
func failOrder(tb TB, got any, op string, bound any, opts []Option) bool {
	tb.Helper()
	tb.Errorf("got %v, want %s %v%s", got, op, bound, annotate(opts))
	return false
}
