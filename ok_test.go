package ok_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stefanvanburen/ok"
)

// recorderTB records failures for asserting on failure messages.
type recorderTB struct {
	helpers int
	errors  []string
}

func (r *recorderTB) Helper() { r.helpers++ }
func (r *recorderTB) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// nopTB discards everything; used by the allocation test and benchmarks.
type nopTB struct{}

func (nopTB) Helper()               {}
func (nopTB) Errorf(string, ...any) {}

// checkFail asserts that exactly one failure was recorded, containing each
// of wants, and that Helper was called.
func checkFail(t *testing.T, r *recorderTB, wants ...string) {
	t.Helper()
	if len(r.errors) != 1 {
		t.Fatalf("recorded %d failures (%q), want 1", len(r.errors), r.errors)
	}
	for _, want := range wants {
		if !strings.Contains(r.errors[0], want) {
			t.Errorf("failure message %q does not contain %q", r.errors[0], want)
		}
	}
	if r.helpers == 0 {
		t.Error("Helper was never called")
	}
}

func checkPass(t *testing.T, r *recorderTB) {
	t.Helper()
	if len(r.errors) != 0 {
		t.Fatalf("recorded failures %q, want none", r.errors)
	}
	if r.helpers == 0 {
		t.Error("Helper was never called")
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.Equal(r, "a", "a") {
		t.Error("Equal returned false for equal values")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.Equal(r, 1, 2) {
		t.Error("Equal returned true for unequal values")
	}
	checkFail(t, r, "got 1, want 2")
}

func TestEqualAmbiguousFormatting(t *testing.T) {
	t.Parallel()
	// int(1) and string "1" both render as "1" with %v; the message must
	// fall back to %#v so the difference is visible.
	r := &recorderTB{}
	if ok.Equal[any](r, 1, "1") {
		t.Error("Equal returned true for unequal values")
	}
	checkFail(t, r, `got 1, want "1"`)
}

func TestNotEqual(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.NotEqual(r, 1, 2) {
		t.Error("NotEqual returned false for unequal values")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.NotEqual(r, "a", "a") {
		t.Error("NotEqual returned true for equal values")
	}
	checkFail(t, r, "got a, want anything else")
}

func TestDeepEqual(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.DeepEqual(r, []int{1, 2}, []int{1, 2}) {
		t.Error("DeepEqual returned false for equal slices")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.DeepEqual(r, []int{1, 2}, []int{1, 3}) {
		t.Error("DeepEqual returned true for unequal slices")
	}
	checkFail(t, r, "not deeply equal", "-want +got")
}

type hidden struct {
	v int
}

func TestDeepEqualUnexportedFields(t *testing.T) {
	t.Parallel()
	// cmp.Diff panics on unexported fields; the failure message must fall
	// back to %#v formatting instead of panicking.
	r := &recorderTB{}
	if ok.DeepEqual(r, hidden{1}, hidden{2}) {
		t.Error("DeepEqual returned true for unequal values")
	}
	checkFail(t, r, "not deeply equal", "ok_test.hidden{v:1}", "ok_test.hidden{v:2}")
}

func TestDeepEqualEmptyDiff(t *testing.T) {
	t.Parallel()
	// Two times differing only in the monotonic clock reading:
	// reflect.DeepEqual reports unequal, but cmp (via time.Time.Equal)
	// produces an empty diff — the message must fall back to %#v.
	t1 := time.Now()
	t2 := t1.Round(0)
	r := &recorderTB{}
	if ok.DeepEqual(r, t1, t2) {
		t.Error("DeepEqual returned true for unequal values")
	}
	checkFail(t, r, "not deeply equal", "got:")
}

func TestCmpEqual(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.CmpEqual(r, []int{1, 2}, []int{1, 2}) {
		t.Error("CmpEqual returned false for equal slices")
	}
	checkPass(t, r)

	// An option changes the meaning of equality: unordered slices compare
	// equal under SortSlices.
	r = &recorderTB{}
	if !ok.CmpEqual(r, []int{3, 1, 2}, []int{1, 2, 3}, cmpopts.SortSlices(func(a, b int) bool { return a < b })) {
		t.Error("CmpEqual returned false for slices equal under SortSlices")
	}
	checkPass(t, r)

	// Options apply to the failure diff too, not just the equality check.
	r = &recorderTB{}
	if ok.CmpEqual(r, []int{3, 1, 2}, []int{1, 2, 4}, cmpopts.SortSlices(func(a, b int) bool { return a < b })) {
		t.Error("CmpEqual returned true for unequal slices")
	}
	checkFail(t, r, "not equal", "-want +got")

	// cmp's panic for uncovered types must propagate: its message names the
	// missing option, which is more useful than a swallowed failure.
	defer func() {
		if recover() == nil {
			t.Error("CmpEqual did not panic on unexported fields without an option")
		}
	}()
	ok.CmpEqual(&recorderTB{}, hidden{1}, hidden{2})
}

func TestCmpEqualIgnoreUnexported(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.CmpEqual(r, hidden{1}, hidden{2}, cmpopts.IgnoreUnexported(hidden{})) {
		t.Error("CmpEqual returned false with IgnoreUnexported covering the difference")
	}
	checkPass(t, r)
}

func TestEqualFunc(t *testing.T) {
	t.Parallel()
	fold := func(a, b string) bool { return strings.EqualFold(a, b) }

	r := &recorderTB{}
	if !ok.EqualFunc(r, "Hello", "hello", fold) {
		t.Error("EqualFunc returned false for equivalent values")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.EqualFunc(r, "Hello", "goodbye", fold) {
		t.Error("EqualFunc returned true for inequivalent values")
	}
	checkFail(t, r, "got Hello, want goodbye")
}

func TestTrue(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.True(r, 1 < 2) {
		t.Error("True returned false")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.True(r, 1 > 2) {
		t.Error("True returned true")
	}
	checkFail(t, r, "got false, want true")
}

func TestTrueMessage(t *testing.T) {
	t.Parallel()
	// A format string and args replace the default failure message.
	r := &recorderTB{}
	got, limit := 2, 3
	if ok.True(r, got > limit, "got %d, want > %d", got, limit) {
		t.Error("True returned true for a false predicate")
	}
	checkFail(t, r, "got 2, want > 3")

	// A passing predicate must not evaluate or report the message.
	r = &recorderTB{}
	if !ok.True(r, got < limit, "got %d, want < %d", got, limit) {
		t.Error("True returned false for a true predicate")
	}
	checkPass(t, r)
}

func TestPanics(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	v, panicked := ok.Panics(r, func() { panic("boom") })
	if !panicked {
		t.Error("Panics returned false for a panicking function")
	}
	// testify's PanicsWithValue: assert on the recovered value.
	ok.Equal(t, v, any("boom"))
	checkPass(t, r)

	r = &recorderTB{}
	if _, panicked := ok.Panics(r, func() {}); panicked {
		t.Error("Panics returned true for a function that returned")
	}
	checkFail(t, r, "function did not panic")
}

func TestNever(t *testing.T) {
	t.Parallel()
	t.Run("stays false", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		if !ok.Never(r, 20*time.Millisecond, 5*time.Millisecond, func(ok.TB) bool {
			attempts++
			return false
		}) {
			t.Error("Never returned false for an always-false condition")
		}
		if attempts < 2 {
			t.Errorf("condition ran %d times, want at least an immediate and a deadline attempt", attempts)
		}
		checkPass(t, r)
	})

	t.Run("becomes true", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		if ok.Never(r, time.Second, time.Millisecond, func(ok.TB) bool {
			attempts++
			return attempts == 3
		}) {
			t.Error("Never returned true for a condition that becomes true")
		}
		if attempts != 3 {
			t.Errorf("condition ran %d times, want to stop at 3 (must fail as soon as satisfied)", attempts)
		}
		checkFail(t, r, "condition satisfied within 1s, want never")
	})
}

func TestNoError(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.NoError(r, nil) {
		t.Error("NoError returned false for nil error")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.NoError(r, errors.New("boom")) {
		t.Error("NoError returned true for non-nil error")
	}
	checkFail(t, r, "unexpected error: boom")
}

func TestError(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.Error(r, errors.New("boom")) {
		t.Error("Error returned false for non-nil error")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.Error(r, nil) {
		t.Error("Error returned true for nil error")
	}
	checkFail(t, r, "got nil, want an error")
}

func TestErrorIs(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	wrapped := fmt.Errorf("context: %w", sentinel)

	r := &recorderTB{}
	if !ok.ErrorIs(r, wrapped, sentinel) {
		t.Error("ErrorIs returned false for wrapped sentinel")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.ErrorIs(r, errors.New("other"), sentinel) {
		t.Error("ErrorIs returned true for unrelated error")
	}
	checkFail(t, r, "got error other, want sentinel in its chain")
}

type codeError struct {
	code int
}

func (e *codeError) Error() string { return fmt.Sprintf("code %d", e.code) }

func TestErrorAs(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("context: %w", &codeError{code: 42})

	r := &recorderTB{}
	ce, found := ok.ErrorAs[*codeError](r, wrapped)
	if !found {
		t.Fatal("ErrorAs returned false for wrapped *codeError")
	}
	if ce.code != 42 {
		t.Errorf("ErrorAs returned error with code %d, want 42", ce.code)
	}
	checkPass(t, r)

	r = &recorderTB{}
	if _, found := ok.ErrorAs[*codeError](r, errors.New("other")); found {
		t.Error("ErrorAs returned true for unrelated error")
	}
	checkFail(t, r, "got error other, want *ok_test.codeError in its chain")
}

func TestZero(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	if !ok.Zero(r, "") {
		t.Error("Zero returned false for zero value")
	}
	checkPass(t, r)

	r = &recorderTB{}
	if ok.Zero(r, 7) {
		t.Error("Zero returned true for non-zero value")
	}
	checkFail(t, r, "got 7, want zero value")
}

func TestEventually(t *testing.T) {
	t.Parallel()
	t.Run("immediate", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		if !ok.Eventually(r, time.Second, time.Millisecond, func(ok.TB) bool {
			attempts++
			return true
		}) {
			t.Error("Eventually returned false for an immediately-true condition")
		}
		if attempts != 1 {
			t.Errorf("condition ran %d times, want 1 (first check must not wait a tick)", attempts)
		}
		checkPass(t, r)
	})

	t.Run("eventually true", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		if !ok.Eventually(r, time.Second, time.Millisecond, func(tb ok.TB) bool {
			attempts++
			return ok.Equal(tb, attempts, 3)
		}) {
			t.Error("Eventually returned false for a condition that becomes true")
		}
		if attempts != 3 {
			t.Errorf("condition ran %d times, want 3", attempts)
		}
		checkPass(t, r)
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		if ok.Eventually(r, 20*time.Millisecond, 5*time.Millisecond, func(tb ok.TB) bool {
			return ok.Equal(tb, 2, 3)
		}) {
			t.Error("Eventually returned true for an always-false condition")
		}
		// Exactly one failure on the enclosing TB: the failing polls along
		// the way must stay silent, and the final attempt's assertion
		// failure must be included in the report.
		checkFail(t, r, "condition not satisfied within 20ms", "got 2, want 3")
	})

	t.Run("timeout without assertions", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		if ok.Eventually(r, time.Millisecond, time.Millisecond, func(ok.TB) bool {
			return false
		}) {
			t.Error("Eventually returned true for an always-false condition")
		}
		checkFail(t, r, "condition not satisfied within 1ms")
	})
}

// TestRealT exercises the passing paths against a real *testing.T.
func TestRealT(t *testing.T) {
	t.Parallel()
	ok.Equal(t, 1+1, 2)
	ok.NotEqual(t, "got", "want")
	ok.DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
	ok.True(t, true)
	ok.NoError(t, nil)
	ok.Error(t, errors.New("boom"))
	ok.Zero(t, time.Duration(0))
}
