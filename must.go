package ok

import "io"

// FatalTB is the subset of [testing.TB] that [Must] requires.
type FatalTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Must returns a TB whose failures are fatal: assertions reporting through
// it halt the test via Fatalf instead of continuing via Errorf. The
// assertions themselves stay non-fatal — the TB carries the policy.
//
//	must := ok.Must(t)
//	u, err := LookupUser("stefan")
//	ok.NoError(must, err)       // halts the test on failure
//	ok.Equal(t, u.Name, "stefan") // continues on failure
//
// As with testing.TB's FailNow, assertions against the returned TB must run
// on the test's goroutine.
func Must(t FatalTB) TB { return mustTB{t} }

type mustTB struct{ t FatalTB }

func (m mustTB) Helper() { m.t.Helper() }

func (m mustTB) Errorf(format string, args ...any) {
	m.t.Helper()
	m.t.Fatalf(format, args...)
}

// Output forwards to the wrapped TB's Output writer when it has one, keeping
// diff color detection intact through the wrapper.
func (m mustTB) Output() io.Writer { return outputWriter(m.t) }
