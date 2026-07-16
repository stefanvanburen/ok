# ok

[![Go Reference](https://pkg.go.dev/badge/github.com/stefanvanburen/ok.svg)](https://pkg.go.dev/github.com/stefanvanburen/ok)

Small, non-fatal test assertions for Go.

```go
func TestUser(t *testing.T) {
    u, err := LookupUser("stefan")
    if !ok.NoError(t, err) {
        return // nothing below makes sense without u
    }
    ok.Equal(t, u.Name, "stefan")
    ok.DeepEqual(t, u.Tags, []string{"go", "running"})
}
```

## Design

**Non-fatal, always.** Every assertion reports failure with `Errorf` and
returns whether it passed. Halting is the *caller's* decision, expressed as
ordinary control flow (`if !ok.NoError(t, err) { return }`) rather than a
hidden `runtime.Goexit`. The `ok.TB` interface doesn't even include `Fatalf`.

**Passing assertions are free.** Equality on comparable types is a `==`
comparison — no reflection, no allocation, ~2ns. [go-cmp] and [colorcmp] are
used only to pretty-print a diff *after* an assertion has failed, where cost
no longer matters.

**Failures name the difference, not the whole value.** `DeepEqual` failures
render a path-based diff via [colorcmp], colored red/green when the test is
run in a terminal (`NO_COLOR` and `FORCE_COLOR` are respected; CI output
stays plain):

```
demo_test.go:21: not deeply equal:
    diff (-want +got):
    Email: -"s@vanburen.xyz" +"stefan@vanburen.xyz"
    Tags[1->?]: -"running"
```

**The type system picks the comparison.** `Equal` requires `comparable` and
uses `==`; slices, maps, and pointer-heavy structs need an explicit
`DeepEqual` (or `EqualFunc` with your own comparator). Using the wrong one is
a compile error, not a surprise at runtime.

## API

| Assertion | Checks |
| --- | --- |
| `Equal[T comparable](tb, got, want)` | `got == want` |
| `NotEqual[T comparable](tb, got, want)` | `got != want` |
| `DeepEqual[T any](tb, got, want)` | `reflect.DeepEqual` |
| `EqualFunc[T any](tb, got, want, equal)` | `equal(got, want)` |
| `True(tb, got)` | `got` |
| `NoError(tb, err)` | `err == nil` |
| `Error(tb, err)` | `err != nil` |
| `ErrorIs(tb, err, target)` | `errors.Is` |
| `ErrorAs[T error](tb, err) (T, bool)` | `errors.As`, returning the match |
| `Zero[T comparable](tb, got)` | `got` is the zero value |

All assertions return `bool` (except `ErrorAs`, which also returns the
matched error).

## Benchmarks

On an Apple M1:

```
BenchmarkEqualInt-8         541268532    2.158 ns/op    0 B/op    0 allocs/op
BenchmarkEqualString-8      416527341    2.886 ns/op    0 B/op    0 allocs/op
BenchmarkDeepEqualSlice-8     9381274    126.3 ns/op   48 B/op    2 allocs/op
BenchmarkDeepEqualMap-8       5407791    222.4 ns/op   64 B/op    6 allocs/op
```

The zero-allocation happy path for `Equal`, `NotEqual`, `EqualFunc`, `True`,
`NoError`, `Error`, `ErrorIs`, and `Zero` is enforced by a test
(`alloc_test.go`), not just promised.

## Prior art

- [akshayjshah/attest] — generics-first, but fatal by default and pays
  go-cmp's reflection cost even when assertions pass.
- [matryer/is] — tiny API; reads test source files on failure to decorate
  messages.
- [nalgeon/be] — non-fatal and minimal; a single `Equal` dispatches at
  runtime (`Equal` method → `bytes.Equal` → `reflect.DeepEqual`), which boxes
  values and defers type mistakes to runtime.
- [shoenig/test] — the closest prior art for splitting comparison strategy by
  constraint (`EqOp` vs `EqFunc`); much larger API, with a parallel fatal
  `must` package.

[go-cmp]: https://github.com/google/go-cmp
[colorcmp]: https://github.com/stefanvanburen/colorcmp
[akshayjshah/attest]: https://github.com/akshayjshah/attest
[matryer/is]: https://github.com/matryer/is
[nalgeon/be]: https://github.com/nalgeon/be
[shoenig/test]: https://github.com/shoenig/test
