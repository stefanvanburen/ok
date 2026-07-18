package ok_test

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.vanburen.xyz/ok"
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
func (nopTB) Fatalf(string, ...any) {}

// checkPass asserts both sides of the passing contract: the assertion
// returned true and recorded nothing.
func checkPass(t *testing.T, r *recorderTB, returned bool) {
	t.Helper()
	if !returned {
		t.Error("assertion returned false, want true")
	}
	if len(r.errors) != 0 {
		t.Fatalf("recorded failures %q, want none", r.errors)
	}
	if r.helpers == 0 {
		t.Error("Helper was never called")
	}
}

// checkFail asserts both sides of the failing contract: the assertion
// returned false and recorded exactly one failure containing each of wants.
func checkFail(t *testing.T, r *recorderTB, returned bool, wants ...string) {
	t.Helper()
	if returned {
		t.Error("assertion returned true, want false")
	}
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

type hidden struct {
	v int
}

func TestAssertions(t *testing.T) {
	t.Parallel()
	fold := func(a, b string) bool { return strings.EqualFold(a, b) }
	less := func(a, b int) bool { return a < b }
	tests := []struct {
		name     string
		assert   func(tb ok.TB) bool
		wantFail []string // substrings of the failure message; nil expects a pass
	}{
		{"Equal pass", func(tb ok.TB) bool { return ok.Equal(tb, "a", "a") }, nil},
		{"Equal fail", func(tb ok.TB) bool { return ok.Equal(tb, 1, 2) }, []string{"got 1, want 2"}},
		// int(1) and string "1" both render as "1" with %v; the message
		// must fall back to %#v so the difference is visible.
		{"Equal ambiguous formatting", func(tb ok.TB) bool { return ok.Equal[any](tb, 1, "1") }, []string{`got 1, want "1"`}},
		{"NotEqual pass", func(tb ok.TB) bool { return ok.NotEqual(tb, 1, 2) }, nil},
		{"NotEqual fail", func(tb ok.TB) bool { return ok.NotEqual(tb, "a", "a") }, []string{"got a, want anything else"}},
		{"DeepEqual pass", func(tb ok.TB) bool { return ok.DeepEqual(tb, []int{1, 2}, []int{1, 2}) }, nil},
		{"DeepEqual fail", func(tb ok.TB) bool { return ok.DeepEqual(tb, []int{1, 2}, []int{1, 3}) }, []string{"not deeply equal", "-want +got"}},
		// cmp.Diff panics on unexported fields; the failure message must
		// fall back to %#v formatting instead of panicking.
		{"DeepEqual unexported fields", func(tb ok.TB) bool { return ok.DeepEqual(tb, hidden{1}, hidden{2}) },
			[]string{"not deeply equal", "ok_test.hidden{v:1}", "ok_test.hidden{v:2}"}},
		// Two times differing only in the monotonic clock reading:
		// reflect.DeepEqual reports unequal, but cmp (via time.Time.Equal)
		// produces an empty diff — the message must fall back to %#v.
		{"DeepEqual empty diff", func(tb ok.TB) bool {
			t1 := time.Now()
			return ok.DeepEqual(tb, t1, t1.Round(0))
		}, []string{"not deeply equal", "got:"}},
		{"CmpEqual pass", func(tb ok.TB) bool { return ok.CmpEqual(tb, []int{1, 2}, []int{1, 2}) }, nil},
		// An option changes the meaning of equality: unordered slices
		// compare equal under SortSlices.
		{"CmpEqual pass under SortSlices", func(tb ok.TB) bool {
			return ok.CmpEqual(tb, []int{3, 1, 2}, []int{1, 2, 3}, cmpopts.SortSlices(less))
		}, nil},
		// Options apply to the failure diff too, not just the equality check.
		{"CmpEqual fail under SortSlices", func(tb ok.TB) bool {
			return ok.CmpEqual(tb, []int{3, 1, 2}, []int{1, 2, 4}, cmpopts.SortSlices(less))
		}, []string{"not equal", "-want +got"}},
		{"CmpEqual pass under IgnoreUnexported", func(tb ok.TB) bool {
			return ok.CmpEqual(tb, hidden{1}, hidden{2}, cmpopts.IgnoreUnexported(hidden{}))
		}, nil},
		{"EqualFunc pass", func(tb ok.TB) bool { return ok.EqualFunc(tb, "Hello", "hello", fold) }, nil},
		{"EqualFunc fail", func(tb ok.TB) bool { return ok.EqualFunc(tb, "Hello", "goodbye", fold) }, []string{"got Hello, want goodbye"}},
		{"True pass", func(tb ok.TB) bool { return ok.True(tb, 1 < 2) }, nil},
		{"True fail", func(tb ok.TB) bool { return ok.True(tb, 1 > 2) }, []string{"got false, want true"}},
		// A format string and args replace the default failure message.
		{"True pass with message", func(tb ok.TB) bool { return ok.True(tb, 2 < 3, "got %d, want < %d", 2, 3) }, nil},
		{"True fail with message", func(tb ok.TB) bool { return ok.True(tb, 2 > 3, "got %d, want > %d", 2, 3) }, []string{"got 2, want > 3"}},
		{"NoError pass", func(tb ok.TB) bool { return ok.NoError(tb, nil) }, nil},
		{"NoError fail", func(tb ok.TB) bool { return ok.NoError(tb, errors.New("boom")) }, []string{"unexpected error: boom"}},
		{"Error pass", func(tb ok.TB) bool { return ok.Error(tb, errors.New("boom")) }, nil},
		{"Error fail", func(tb ok.TB) bool { return ok.Error(tb, nil) }, []string{"got nil, want an error"}},
		{"ErrorIs pass", func(tb ok.TB) bool {
			sentinel := errors.New("sentinel")
			return ok.ErrorIs(tb, fmt.Errorf("context: %w", sentinel), sentinel)
		}, nil},
		{"ErrorIs fail", func(tb ok.TB) bool {
			return ok.ErrorIs(tb, errors.New("other"), errors.New("sentinel"))
		}, []string{"got error other, want sentinel in its chain"}},
		{"Zero pass", func(tb ok.TB) bool { return ok.Zero(tb, "") }, nil},
		{"Zero fail", func(tb ok.TB) bool { return ok.Zero(tb, 7) }, []string{"got 7, want zero value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &recorderTB{}
			returned := tt.assert(r)
			if tt.wantFail == nil {
				checkPass(t, r, returned)
			} else {
				checkFail(t, r, returned, tt.wantFail...)
			}
		})
	}
}

func TestCmpEqualPanics(t *testing.T) {
	t.Parallel()
	// cmp's panic for uncovered types must propagate: its message names the
	// missing option, which is more useful than a swallowed failure.
	defer func() {
		if recover() == nil {
			t.Error("CmpEqual did not panic on unexported fields without an option")
		}
	}()
	ok.CmpEqual(&recorderTB{}, hidden{1}, hidden{2})
}

type codeError struct {
	code int
}

func (e *codeError) Error() string { return fmt.Sprintf("code %d", e.code) }

func TestErrorAs(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	ce, found := ok.ErrorAs[*codeError](r, fmt.Errorf("context: %w", &codeError{code: 42}))
	checkPass(t, r, found)
	if found && ce.code != 42 {
		t.Errorf("ErrorAs returned error with code %d, want 42", ce.code)
	}

	r = &recorderTB{}
	_, found = ok.ErrorAs[*codeError](r, errors.New("other"))
	checkFail(t, r, found, "got error other, want *ok_test.codeError in its chain")
}

func TestPanics(t *testing.T) {
	t.Parallel()
	r := &recorderTB{}
	v, panicked := ok.Panics(r, func() { panic("boom") })
	checkPass(t, r, panicked)
	// testify's PanicsWithValue: assert on the recovered value.
	ok.Equal(t, v, any("boom"))

	r = &recorderTB{}
	_, panicked = ok.Panics(r, func() {})
	checkFail(t, r, panicked, "function did not panic")
}

func TestEventually(t *testing.T) {
	t.Parallel()
	t.Run("immediate", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		returned := ok.Eventually(r, time.Second, time.Millisecond, func(ok.TB) bool {
			attempts++
			return true
		})
		checkPass(t, r, returned)
		if attempts != 1 {
			t.Errorf("condition ran %d times, want 1 (first check must not wait a tick)", attempts)
		}
	})

	t.Run("eventually true", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		returned := ok.Eventually(r, time.Second, time.Millisecond, func(tb ok.TB) bool {
			attempts++
			return ok.Equal(tb, attempts, 3)
		})
		checkPass(t, r, returned)
		if attempts != 3 {
			t.Errorf("condition ran %d times, want 3", attempts)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		returned := ok.Eventually(r, 20*time.Millisecond, 5*time.Millisecond, func(tb ok.TB) bool {
			return ok.Equal(tb, 2, 3)
		})
		// Exactly one failure on the enclosing TB: the failing polls along
		// the way must stay silent, and the final attempt's assertion
		// failure must be included in the report.
		checkFail(t, r, returned, "condition not satisfied within 20ms", "got 2, want 3")
	})

	t.Run("timeout without assertions", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		returned := ok.Eventually(r, time.Millisecond, time.Millisecond, func(ok.TB) bool {
			return false
		})
		checkFail(t, r, returned, "condition not satisfied within 1ms")
	})
}

func TestNever(t *testing.T) {
	t.Parallel()
	t.Run("stays false", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		returned := ok.Never(r, 20*time.Millisecond, 5*time.Millisecond, func(ok.TB) bool {
			attempts++
			return false
		})
		checkPass(t, r, returned)
		if attempts < 2 {
			t.Errorf("condition ran %d times, want at least an immediate and a deadline attempt", attempts)
		}
	})

	t.Run("becomes true", func(t *testing.T) {
		t.Parallel()
		r := &recorderTB{}
		attempts := 0
		returned := ok.Never(r, time.Second, time.Millisecond, func(ok.TB) bool {
			attempts++
			return attempts == 3
		})
		checkFail(t, r, returned, "condition satisfied within 1s, want never")
		if attempts != 3 {
			t.Errorf("condition ran %d times, want to stop at 3 (must fail as soon as satisfied)", attempts)
		}
	})
}

// fatalTB records Fatalf calls and then halts the goroutine, modelling how
// testing.T.Fatalf ends a test via runtime.Goexit.
type fatalTB struct {
	helpers int
	fatals  []string
}

func (f *fatalTB) Helper() { f.helpers++ }
func (f *fatalTB) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
	runtime.Goexit()
}

// ran reports whether fn returned normally rather than halting via Goexit.
// fn runs in its own goroutine so a Fatalf/Goexit can't take down the test.
func ran(fn func()) (returned bool) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
		returned = true
	}()
	<-done
	return returned
}

func TestMustNoError(t *testing.T) {
	t.Parallel()

	f := &fatalTB{}
	if !ran(func() { ok.MustNoError(f, nil) }) {
		t.Error("MustNoError halted on a nil error")
	}
	if len(f.fatals) != 0 {
		t.Fatalf("nil error reported fatal failures %q", f.fatals)
	}

	f = &fatalTB{}
	if ran(func() { ok.MustNoError(f, errors.New("boom")) }) {
		t.Error("MustNoError did not halt on a non-nil error")
	}
	if len(f.fatals) != 1 || !strings.Contains(f.fatals[0], "unexpected error: boom") {
		t.Errorf("fatals = %q, want exactly one containing %q", f.fatals, "unexpected error: boom")
	}
	if f.helpers == 0 {
		t.Error("Helper was never called")
	}
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
