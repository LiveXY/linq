package linq

import (
	"strings"
	"testing"
)

// Helper to generate a large slice
func makeRange(min, max int) []int {
	a := make([]int, max-min)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func BenchmarkFromSlice(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		From(data).ToSlice()
	}
}

func BenchmarkWhere(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		From(data).Where(func(i int) bool { return i%2 == 0 }).ToSlice()
	}
}

func BenchmarkSelect(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Select(From(data), func(i int) int { return i * 2 }).ToSlice()
	}
}

func BenchmarkMinBy(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MinBy(From(data), func(i int) int { return i })
	}
}

func BenchmarkGroupBy(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupBy(From(data), func(i int) int { return i % 100 }).ToSlice()
	}
}

func BenchmarkFromString(b *testing.B) {
	// A string with mixed ASCII and Unicode
	str := strings.Repeat("a", 1000) + strings.Repeat("世", 1000) + strings.Repeat("🌍", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromString(str).Count()
	}
}

func BenchmarkUnion(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Union(data1, data2)
	}
}

func BenchmarkSort(b *testing.B) {
	data := makeRange(0, 1000)
	// Just reverse it to give sort some work
	for i := 0; i < len(data)/2; i++ {
		data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OrderBy(From(data), func(i int) int { return i }).ToSlice()
	}
}
