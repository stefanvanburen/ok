package ok_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.vanburen.xyz/ok"
)

func TestMoreAssertions(t *testing.T) {
	t.Parallel()

	var nilPtr *int
	var nilErrPtr *os.PathError
	var nilIface error = nilErrPtr

	tests := []struct {
		name     string
		assert   func(tb ok.TB) bool
		wantFail []string
	}{
		{"Nil untyped", func(tb ok.TB) bool { return ok.Nil(tb, nil) }, nil},
		{"Nil pointer", func(tb ok.TB) bool { return ok.Nil(tb, nilPtr) }, nil},
		{"Nil slice", func(tb ok.TB) bool { return ok.Nil(tb, []int(nil)) }, nil},
		// == nil says no to an interface holding a nil pointer; Nil says yes.
		{"Nil interface holding nil pointer", func(tb ok.TB) bool { return ok.Nil(tb, nilIface) }, nil},
		{"Nil fail", func(tb ok.TB) bool { return ok.Nil(tb, 7) }, []string{"got 7, want nil"}},
		{"Nil fail with message", func(tb ok.TB) bool {
			return ok.Nil(tb, errors.New("boom"), ok.Sprintf("step %d", 2))
		}, []string{"got boom, want nil: step 2"}},
		{"NotNil pass", func(tb ok.TB) bool { return ok.NotNil(tb, errors.New("boom")) }, nil},
		{"NotNil pass on a non-nilable value", func(tb ok.TB) bool { return ok.NotNil(tb, 0) }, nil},
		{"NotNil fail", func(tb ok.TB) bool { return ok.NotNil(tb, nilPtr) }, []string{"got nil, want non-nil"}},
		{"Len pass", func(tb ok.TB) bool { return ok.Len(tb, []string{"a", "b"}, 2) }, nil},
		{"Len fail shows the elements", func(tb ok.TB) bool {
			return ok.Len(tb, []string{"a", "b", "c"}, 2)
		}, []string{"got 3 elements [a b c], want 2"}},
		{"Greater pass", func(tb ok.TB) bool { return ok.Greater(tb, 3, 2) }, nil},
		{"Greater fail", func(tb ok.TB) bool { return ok.Greater(tb, 2, 2) }, []string{"got 2, want > 2"}},
		{"GreaterOrEqual pass", func(tb ok.TB) bool { return ok.GreaterOrEqual(tb, 2, 2) }, nil},
		{"GreaterOrEqual fail", func(tb ok.TB) bool {
			return ok.GreaterOrEqual(tb, 1, 2, ok.Sprintf("rows"))
		}, []string{"got 1, want >= 2: rows"}},
		{"Less pass", func(tb ok.TB) bool { return ok.Less(tb, "a", "b") }, nil},
		{"Less fail", func(tb ok.TB) bool { return ok.Less(tb, 2.5, 1.5) }, []string{"got 2.5, want < 1.5"}},
		{"LessOrEqual pass", func(tb ok.TB) bool { return ok.LessOrEqual(tb, 2, 2) }, nil},
		{"LessOrEqual fail", func(tb ok.TB) bool { return ok.LessOrEqual(tb, 3, 2) }, []string{"got 3, want <= 2"}},
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

func TestMust(t *testing.T) {
	t.Parallel()

	f := &fatalTB{}
	must := ok.Must(f)
	if !ran(func() { ok.Equal(must, 1, 1) }) {
		t.Error("Must halted on a passing assertion")
	}
	if len(f.fatals) != 0 {
		t.Fatalf("passing assertion reported %q", f.fatals)
	}

	f = &fatalTB{}
	must = ok.Must(f)
	if ran(func() { ok.Equal(must, 1, 2) }) {
		t.Error("Must did not halt on a failing assertion")
	}
	if len(f.fatals) != 1 || !strings.Contains(f.fatals[0], "got 1, want 2") {
		t.Errorf("fatals = %q, want exactly one containing %q", f.fatals, "got 1, want 2")
	}
	if f.helpers == 0 {
		t.Error("Helper was never called")
	}
}

// A failure through Must is attributed to the test's own line, not to one
// inside ok: Helper has to reach testing.T without a frame of Must's own in
// between. Only a real failing *testing.T shows the attribution, so the test
// runs itself in a child process to fail there.
func TestMustReportsTheCallerLine(t *testing.T) {
	if os.Getenv("OK_MUST_CHILD") == "1" {
		must := ok.Must(t)
		ok.Equal(must, 1, 2) // the line the report must name
		t.Error("Must did not halt")

		return
	}

	t.Parallel()

	cmd := exec.Command(os.Args[0], "-test.run=^TestMustReportsTheCallerLine$", "-test.v")
	cmd.Env = append(os.Environ(), "OK_MUST_CHILD=1")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child passed; output:\n%s", out)
	}

	want := fmt.Sprintf("more_test.go:%d: got 1, want 2", markedLine(t, "the line the report must name"))
	if !strings.Contains(string(out), want) {
		t.Errorf("child output does not report %q:\n%s", want, out)
	}

	if strings.Contains(string(out), "Must did not halt") {
		t.Errorf("child kept going after the failure:\n%s", out)
	}
}

// markedLine is the line of this file whose comment ends with marker.
func markedLine(t *testing.T, marker string) int {
	t.Helper()

	src, err := os.ReadFile("more_test.go")
	ok.MustNoError(t, err)

	for i, line := range strings.Split(string(src), "\n") {
		if strings.HasSuffix(line, "// "+marker) {
			return i + 1
		}
	}

	t.Fatalf("no line marked %q", marker)

	return 0
}
