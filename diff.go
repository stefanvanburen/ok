package ok

import (
	"fmt"
	"io"

	"github.com/google/go-cmp/cmp"
	"github.com/stefanvanburen/colorcmp"
)

// diff pretty-prints the difference between got and want. It is only called
// after an assertion has failed, so its cost never taxes a passing test.
func diff[T any](tb TB, got, want T) (out string) {
	defer func() {
		// cmp panics on types it can't compare (e.g. structs with
		// unexported fields); fall back to plain formatting.
		if recover() != nil {
			out = fallback(got, want)
		}
	}()
	// *testing.T (Go 1.25+) provides Output; colorcmp uses the writer for
	// color detection. TB implementations without it get no colors.
	var w io.Writer
	if o, ok := tb.(interface{ Output() io.Writer }); ok {
		w = o.Output()
	}
	r := colorcmp.New(w)
	if cmp.Equal(want, got, cmp.Reporter(r)) {
		return fallback(got, want)
	}
	if d := r.String(); d != "" {
		return "diff (-want +got):\n" + d
	}
	return fallback(got, want)
}

func fallback[T any](got, want T) string {
	return fmt.Sprintf("got:  %#v\nwant: %#v", got, want)
}
