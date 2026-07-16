package ok_test

import (
	"testing"

	"github.com/stefanvanburen/ok"
)

func BenchmarkEqualInt(b *testing.B) {
	tb := nopTB{}
	b.ReportAllocs()
	for b.Loop() {
		ok.Equal(tb, 42, 42)
	}
}

func BenchmarkEqualString(b *testing.B) {
	tb := nopTB{}
	b.ReportAllocs()
	for b.Loop() {
		ok.Equal(tb, "hello, world", "hello, world")
	}
}

func BenchmarkDeepEqualSlice(b *testing.B) {
	tb := nopTB{}
	got := []int{1, 2, 3, 4, 5}
	want := []int{1, 2, 3, 4, 5}
	b.ReportAllocs()
	for b.Loop() {
		ok.DeepEqual(tb, got, want)
	}
}

func BenchmarkCmpEqualSlice(b *testing.B) {
	tb := nopTB{}
	got := []int{1, 2, 3, 4, 5}
	want := []int{1, 2, 3, 4, 5}
	b.ReportAllocs()
	for b.Loop() {
		ok.CmpEqual(tb, got, want)
	}
}

func BenchmarkDeepEqualMap(b *testing.B) {
	tb := nopTB{}
	got := map[string]int{"a": 1, "b": 2}
	want := map[string]int{"a": 1, "b": 2}
	b.ReportAllocs()
	for b.Loop() {
		ok.DeepEqual(tb, got, want)
	}
}
