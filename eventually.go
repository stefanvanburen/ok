package ok

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Eventually asserts that attempt returns true within waitFor, polling every
// tick. The first attempt runs immediately, and a final attempt runs at the
// deadline.
//
// Each attempt receives a fresh TB that buffers failures: polls that fail
// while waiting stay silent, and if the condition is never satisfied, the
// failures from the final attempt are reported alongside the timeout.
// Because assertions return bool, they compose as the condition itself:
//
//	ok.Eventually(t, 5*time.Second, 10*time.Millisecond, func(tb ok.TB) bool {
//		n, err := store.Count(ctx)
//		return ok.NoError(tb, err) && ok.Equal(tb, n, 3)
//	})
//
// Report failures inside attempt to its tb parameter, not the enclosing
// test's: asserting against the enclosing *testing.T would report a failure
// on every poll.
func Eventually(tb TB, waitFor, tick time.Duration, attempt func(tb TB) bool) bool {
	tb.Helper()
	satisfied, last := poll(tb, waitFor, tick, attempt)
	if satisfied {
		return true
	}
	msg := fmt.Sprintf("condition not satisfied within %v", waitFor)
	if len(last.failures) > 0 {
		msg += "; failures from final attempt:\n" + strings.Join(last.failures, "\n")
	}
	tb.Errorf("%s", msg)
	return false
}

// Never asserts that attempt returns false for the entire waitFor window,
// polling every tick — [Eventually]'s inverse. Like Eventually, the first
// attempt runs immediately, a final attempt runs at the deadline, and
// failures reported inside attempt are buffered, never surfaced: an attempt
// is simply satisfied or not.
func Never(tb TB, waitFor, tick time.Duration, attempt func(tb TB) bool) bool {
	tb.Helper()
	if satisfied, _ := poll(tb, waitFor, tick, attempt); satisfied {
		tb.Errorf("condition satisfied within %v, want never", waitFor)
		return false
	}
	return true
}

// poll runs attempt every tick until it returns true or waitFor elapses,
// reporting whether it ever returned true and the final attempt's recorder.
func poll(tb TB, waitFor, tick time.Duration, attempt func(tb TB) bool) (satisfied bool, last *recorder) {
	deadline := time.Now().Add(waitFor)
	for {
		rec := &recorder{outer: tb}
		if attempt(rec) {
			return true, rec
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false, rec
		}
		time.Sleep(min(tick, remaining))
	}
}

// recorder is the TB handed to each Eventually attempt: it buffers failures
// so that only the final attempt's are ever reported.
type recorder struct {
	outer    TB
	failures []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

// Output forwards to the enclosing TB's Output writer when it has one, so
// diffs rendered inside an attempt keep the same color detection as diffs
// rendered directly against *testing.T.
func (r *recorder) Output() io.Writer {
	if o, ok := r.outer.(interface{ Output() io.Writer }); ok {
		return o.Output()
	}
	return nil
}
