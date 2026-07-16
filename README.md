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

**The name picks the comparison.** `Equal` requires `comparable` and uses
`==`; slices, maps, and pointer-heavy structs need an explicit `DeepEqual`,
`CmpEqual` (when equality needs [go-cmp] options, e.g. `protocmp.Transform`
for protobufs), or `EqualFunc` with your own comparator. Using the wrong one
is a compile error, not a surprise at runtime — and each escalation's cost
is in its name, never in a default.

## API

| Assertion | Checks |
| --- | --- |
| `Equal[T comparable](tb, got, want)` | `got == want` |
| `NotEqual[T comparable](tb, got, want)` | `got != want` |
| `DeepEqual[T any](tb, got, want)` | `reflect.DeepEqual` |
| `CmpEqual[T any](tb, got, want, opts...)` | `cmp.Equal` with [go-cmp] options |
| `EqualFunc[T any](tb, got, want, equal)` | `equal(got, want)` |
| `True(tb, got, msgAndArgs...)` | `got`, with an optional formatted failure message |
| `Panics(tb, f) (any, bool)` | `f` panics; returns the recovered value |
| `NoError(tb, err)` | `err == nil` |
| `Error(tb, err)` | `err != nil` |
| `ErrorIs(tb, err, target)` | `errors.Is` |
| `ErrorAs[T error](tb, err) (T, bool)` | `errors.As`, returning the match |
| `Zero[T comparable](tb, got)` | `got` is the zero value |
| `Eventually(tb, waitFor, tick, attempt)` | `attempt` returns true within `waitFor` |
| `Never(tb, waitFor, tick, attempt)` | `attempt` stays false throughout `waitFor` |

All assertions return `bool` (except `ErrorAs`, which also returns the
matched error).

Because assertions return `bool`, `Eventually` needs no second
`EventuallyWithT`-style variant: assertions *are* the condition, and the
`TB` handed to each attempt buffers their failures so only the final
attempt's are reported on timeout.

```go
ok.Eventually(t, 5*time.Second, 10*time.Millisecond, func(tb ok.TB) bool {
    n, err := store.Count(ctx)
    return ok.NoError(tb, err) && ok.Equal(tb, n, 3)
})
```

## Cookbook

The API is deliberately small; most of testify's surface is a stdlib call
away. (Note testify takes `(t, expected, actual)`; ok takes `(tb, got,
want)`.)

| testify | with ok |
| --- | --- |
| `assert.Same(t, want, got)` | `ok.Equal(t, got, want)` — `==` on pointers is identity |
| `assert.Nil(t, p)` | `ok.Equal(t, p, nil)` |
| `assert.EqualValues(t, 3, count)` | `ok.Equal(t, int(count), 3)` — explicit conversion |
| `assert.Len(t, s, 2)` | `ok.Equal(t, len(s), 2)` |
| `assert.Empty(t, s)` | `ok.Zero(t, len(s))` |
| `assert.Contains(t, s, v)` | `ok.True(t, slices.Contains(s, v))` (or `strings.Contains`) |
| `assert.ElementsMatch(t, a, b)` | `ok.CmpEqual(t, a, b, cmpopts.SortSlices(less))` |
| `assert.InDelta(t, want, got, 0.01)` | `ok.CmpEqual(t, got, want, cmpopts.EquateApprox(0, 0.01))` |
| `assert.WithinDuration(t, a, b, d)` | `ok.CmpEqual(t, a, b, cmpopts.EquateApproxTime(d))` |
| `assert.JSONEq(t, want, got)` | unmarshal both into `any`, then `ok.DeepEqual` (see below) |
| `assert.Greater(t, a, b)` | `ok.True(t, a > b, "got %d, want > %d", a, b)` |
| `assert.Regexp(t, re, s)` | `ok.True(t, regexp.MustCompile(re).MatchString(s))` |
| `assert.ErrorContains(t, err, "x")` | `ok.Error(t, err)` then `ok.True(t, strings.Contains(err.Error(), "x"))` |
| `assert.FileExists(t, p)` | `_, err := os.Stat(p); ok.NoError(t, err)` |
| `assert.Panics(t, fn)` | `ok.Panics(t, fn)` |
| `assert.PanicsWithValue(t, v, fn)` | `got, _ := ok.Panics(t, fn)` then assert on `got` |
| `assert.Never(t, cond, wait, tick)` | `ok.Never(t, wait, tick, attempt)` |
| `require.*` | `if !ok.X(…) { return }` (or `t.FailNow()`) |

The JSONEq translation is worth spelling out, because the failure diff names
the path to each difference inside the JSON rather than dumping both
documents:

```go
var got, want any
ok.NoError(t, json.Unmarshal(gotBody, &got))
ok.NoError(t, json.Unmarshal([]byte(`{"name":"stefan","retries":5}`), &want))
ok.DeepEqual(t, got, want)
```

```
not deeply equal:
diff (-want +got):
["retries"].(float64): -5 +3
```

## Benchmarks

On an Apple M1:

```
BenchmarkEqualInt-8         541268532    2.158 ns/op    0 B/op    0 allocs/op
BenchmarkEqualString-8      416527341    2.886 ns/op    0 B/op    0 allocs/op
BenchmarkDeepEqualSlice-8     9381274    126.3 ns/op   48 B/op    2 allocs/op
BenchmarkDeepEqualMap-8       5407791    222.4 ns/op   64 B/op    6 allocs/op
BenchmarkCmpEqualSlice-8       249854     4809 ns/op 1504 B/op   16 allocs/op
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
