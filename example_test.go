package ok_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.vanburen.xyz/ok"
)

// printTB stands in for *testing.T so the examples below can show real
// failure messages.
type printTB struct{}

func (printTB) Helper()                           {}
func (printTB) Errorf(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func ExampleEqual() {
	var t printTB // in real tests: *testing.T
	ok.Equal(t, "hello", "hello")
	ok.Equal(t, 1, 2)
	// Output:
	// got 1, want 2
}

func ExampleEventually() {
	var t printTB // in real tests: *testing.T
	tries := 0
	ok.Eventually(t, time.Second, time.Millisecond, func(tb ok.TB) bool {
		tries++
		return ok.Equal(tb, tries, 3)
	})
	fmt.Println("tries:", tries)
	// Output:
	// tries: 3
}

// ExampleCmpEqual asserts order-insensitive slice equality, testify's
// assert.ElementsMatch.
func ExampleCmpEqual() {
	var t printTB // in real tests: *testing.T
	less := func(a, b int) bool { return a < b }
	if ok.CmpEqual(t, []int{3, 1, 2}, []int{1, 2, 3}, cmpopts.SortSlices(less)) {
		fmt.Println("elements match")
	}
	// Output:
	// elements match
}

// ExampleDeepEqual_json compares two JSON documents structurally, testify's
// assert.JSONEq: on failure, the diff names the path to each difference.
func ExampleDeepEqual_json() {
	var t printTB // in real tests: *testing.T
	var got, want any
	ok.NoError(t, json.Unmarshal([]byte(`{"b":2,"a":1}`), &got))
	ok.NoError(t, json.Unmarshal([]byte(`{"a":1,"b":2}`), &want))
	if ok.DeepEqual(t, got, want) {
		fmt.Println("same JSON")
	}
	// Output:
	// same JSON
}

func ExampleNoError() {
	var t printTB // in real tests: *testing.T
	var err error
	if !ok.NoError(t, err) {
		return // stop: nothing below makes sense without the value
	}
	fmt.Println("continuing")
	// Output:
	// continuing
}
