# ok

[![Go Reference](https://pkg.go.dev/badge/github.com/stefanvanburen/ok.svg)](https://pkg.go.dev/github.com/stefanvanburen/ok)

Small test assertions for Go.

```go
func TestUser(t *testing.T) {
    u, err := LookupUser("stefan")
    ok.MustNoError(t, err)

    ok.Equal(t, u.Name, "stefan")
    ok.DeepEqual(t, u.Tags, []string{"go", "running"})
}
```

## Installation

```console
$ go get github.com/stefanvanburen/ok
```

## Usage

Assertions report failures with `Errorf` and return whether they passed, so
a test decides for itself when to stop:

```go
if !ok.DeepEqual(t, got, want) {
    return
}
```

The exception is `MustNoError`, which calls `Fatalf`: when a test can't get
a value it needs, there's rarely a point in continuing.

There are four equality assertions. `Equal` works on `comparable` types and
uses `==`. `DeepEqual` uses `reflect.DeepEqual`. `CmpEqual` uses [go-cmp]
and accepts its options (`protocmp.Transform` for protobuf messages,
`cmpopts.EquateApprox` for floats, and so on). `EqualFunc` takes a
comparison function. Handing a slice to `Equal` is a compile error rather
than a runtime surprise, and the common case stays cheap: a passing `Equal`
is a `==` comparison, with go-cmp and [colorcmp] running only after a
failure, to format the diff. A test enforces that passing assertions don't
allocate (`DeepEqual`, `CmpEqual`, and `ErrorAs` excepted; they use
reflection).

Failed deep comparisons print a diff naming the path to each difference,
colored when the output is a terminal:

```
user_test.go:21: not deeply equal:
    diff (-want +got):
    Email: -"s@vanburen.xyz" +"stefan@vanburen.xyz"
    Tags[1->?]: -"running"
```

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
| `MustNoError(tb, err)` | `err == nil`, halting the test otherwise |
| `Error(tb, err)` | `err != nil` |
| `ErrorIs(tb, err, target)` | `errors.Is` |
| `ErrorAs[T error](tb, err) (T, bool)` | `errors.As`, returning the match |
| `Zero[T comparable](tb, got)` | `got` is the zero value |
| `Eventually(tb, waitFor, tick, attempt)` | `attempt` returns true within `waitFor` |
| `Never(tb, waitFor, tick, attempt)` | `attempt` stays false throughout `waitFor` |

`Eventually` and `Never` poll a condition. Assertions can serve as the
condition, since they return their result; failures inside an attempt are
buffered, and only the final attempt's are reported on timeout:

```go
ok.Eventually(t, 5*time.Second, 10*time.Millisecond, func(tb ok.TB) bool {
    n, err := store.Count(ctx)
    return ok.NoError(tb, err) && ok.Equal(tb, n, 3)
})
```

## Cookbook

Much of testify's surface is a stdlib call away. (testify takes
`(t, expected, actual)`; ok takes `(tb, got, want)`.)

| testify | with ok |
| --- | --- |
| `assert.Same(t, want, got)` | `ok.Equal(t, got, want)` (`==` on pointers is identity) |
| `assert.Nil(t, p)` | `ok.Equal(t, p, nil)` |
| `assert.EqualValues(t, 3, count)` | `ok.Equal(t, int(count), 3)` |
| `assert.Len(t, s, 2)` | `ok.Equal(t, len(s), 2)` |
| `assert.Empty(t, s)` | `ok.Zero(t, len(s))` |
| `assert.Contains(t, s, v)` | `ok.True(t, slices.Contains(s, v))` |
| `assert.ElementsMatch(t, a, b)` | `ok.CmpEqual(t, a, b, cmpopts.SortSlices(less))` |
| `assert.InDelta(t, want, got, 0.01)` | `ok.CmpEqual(t, got, want, cmpopts.EquateApprox(0, 0.01))` |
| `assert.WithinDuration(t, a, b, d)` | `ok.CmpEqual(t, a, b, cmpopts.EquateApproxTime(d))` |
| `assert.JSONEq(t, want, got)` | unmarshal both into `any`, then `ok.DeepEqual` |
| `assert.Greater(t, a, b)` | `ok.True(t, a > b, "got %d, want > %d", a, b)` |
| `assert.Regexp(t, re, s)` | `ok.True(t, regexp.MustCompile(re).MatchString(s))` |
| `assert.ErrorContains(t, err, "x")` | `ok.True(t, strings.Contains(err.Error(), "x"))` |
| `assert.FileExists(t, p)` | `_, err := os.Stat(p); ok.NoError(t, err)` |
| `assert.Panics(t, fn)` | `ok.Panics(t, fn)` |
| `assert.PanicsWithValue(t, v, fn)` | `got, _ := ok.Panics(t, fn)`, then assert on `got` |
| `assert.Never(t, cond, wait, tick)` | `ok.Never(t, wait, tick, attempt)` |
| `require.NoError(t, err)` | `ok.MustNoError(t, err)` |
| other `require.*` | `if !ok.X(…) { return }` |

The JSONEq translation gets a diff into the JSON structure, rather than a
dump of both documents:

```go
var got, want any
ok.MustNoError(t, json.Unmarshal(gotBody, &got))
ok.MustNoError(t, json.Unmarshal([]byte(`{"name":"stefan","retries":5}`), &want))
ok.DeepEqual(t, got, want)
```

```
not deeply equal:
diff (-want +got):
["retries"].(float64): -5 +3
```

## Prior art

- [akshayjshah/attest]: generics and a small API, but fatal by default, and
  go-cmp runs on every comparison.
- [matryer/is], [nalgeon/be]: small non-fatal assertion sets.
- [shoenig/test]: splits equality assertions by comparison strategy.

[go-cmp]: https://github.com/google/go-cmp
[colorcmp]: https://github.com/stefanvanburen/colorcmp
[akshayjshah/attest]: https://github.com/akshayjshah/attest
[matryer/is]: https://github.com/matryer/is
[nalgeon/be]: https://github.com/nalgeon/be
[shoenig/test]: https://github.com/shoenig/test
