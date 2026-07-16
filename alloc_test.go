// The race detector instruments allocations, so the counts below only hold
// without it.
//go:build !race

package ok_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stefanvanburen/ok"
)

// TestPassingAssertionsDoNotAllocate enforces the package's core contract:
// assertions that pass are free. DeepEqual and ErrorAs are excluded — both
// are documented to use reflection.
func TestPassingAssertionsDoNotAllocate(t *testing.T) {
	tb := nopTB{}
	sentinel := errors.New("sentinel")
	wrapped := fmt.Errorf("context: %w", sentinel)
	intsEqual := func(a, b int) bool { return a == b }

	allocs := testing.AllocsPerRun(1000, func() {
		ok.Equal(tb, 42, 42)
		ok.Equal(tb, "go", "go")
		ok.NotEqual(tb, 1, 2)
		ok.EqualFunc(tb, 1, 1, intsEqual)
		ok.True(tb, true)
		ok.NoError(tb, nil)
		ok.Error(tb, sentinel)
		ok.ErrorIs(tb, wrapped, sentinel)
		ok.Zero(tb, 0)
	})
	if allocs != 0 {
		t.Errorf("passing assertions allocated %v times per run, want 0", allocs)
	}
}
