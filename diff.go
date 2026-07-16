package ok

import (
	"fmt"
	"io"
	"slices"

	"github.com/google/go-cmp/cmp"
	"github.com/stefanvanburen/colorcmp"
)

// outputWriter returns tb's Output writer when it has one — *testing.T
// (Go 1.25+) does — for colorcmp's color detection. All probes of this
// optional capability go through here so they can't drift apart. The
// parameter is any rather than TB so wrappers holding narrower interfaces
// (mustTB's FatalTB) can probe too.
func outputWriter(tb any) io.Writer {
	if o, ok := tb.(interface{ Output() io.Writer }); ok {
		return o.Output()
	}
	return nil
}

// diff pretty-prints the difference between got and want. It is only called
// after an assertion has failed, so its cost never taxes a passing test.
func diff[T any](tb TB, got, want T, opts ...cmp.Option) (out string) {
	defer func() {
		// cmp panics on types it can't compare (e.g. structs with
		// unexported fields); fall back to plain formatting.
		if recover() != nil {
			out = fallback(got, want)
		}
	}()
	r := colorcmp.New(outputWriter(tb))
	opts = append(slices.Clip(opts), cmp.Reporter(r))
	if cmp.Equal(want, got, opts...) {
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
