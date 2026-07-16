package ok

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
)

// diff pretty-prints the difference between got and want. It is only called
// after an assertion has failed, so its cost never taxes a passing test.
func diff[T any](got, want T) (out string) {
	defer func() {
		// cmp.Diff panics on types it can't compare (e.g. structs with
		// unexported fields); fall back to plain formatting.
		if recover() != nil {
			out = fallback(got, want)
		}
	}()
	if d := cmp.Diff(want, got); d != "" {
		return "diff (-want +got):\n" + d
	}
	return fallback(got, want)
}

func fallback[T any](got, want T) string {
	return fmt.Sprintf("got:  %#v\nwant: %#v", got, want)
}
