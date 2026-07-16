package ok_test

import (
	"fmt"
	"time"

	"github.com/stefanvanburen/ok"
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
