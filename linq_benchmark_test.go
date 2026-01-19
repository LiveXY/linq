package linq

import (
	"strings"
	"testing"
)

// 辅助函数：生成大切片
func makeRange(min, max int) []int {
	a := make([]int, max-min)
	for i := range a {
		a[i] = min + i
	}
	return a
}

// BenchmarkFromSlice 基准测试：从切片创建查询并还原
func BenchmarkFromSlice(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		From(data).ToSlice()
	}
}

// BenchmarkWhere 基准测试：过滤操作
func BenchmarkWhere(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		From(data).Where(func(i int) bool { return i%2 == 0 }).ToSlice()
	}
}

// BenchmarkSelect 基准测试：映射操作
func BenchmarkSelect(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Select(From(data), func(i int) int { return i * 2 }).ToSlice()
	}
}

// BenchmarkMinBy 基准测试：按条件查找最小值
func BenchmarkMinBy(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MinBy(From(data), func(i int) int { return i })
	}
}

// BenchmarkGroupBy 基准测试：分组操作
func BenchmarkGroupBy(b *testing.B) {
	data := makeRange(0, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupBy(From(data), func(i int) int { return i % 100 }).ToSlice()
	}
}

// BenchmarkFromString 基准测试：从字符串创建查询
func BenchmarkFromString(b *testing.B) {
	// 包含 ASCII 和 Unicode 的混合字符串
	str := strings.Repeat("a", 1000) + strings.Repeat("世", 1000) + strings.Repeat("🌍", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromString(str).Count()
	}
}

// BenchmarkUnion 基准测试：集合并集
func BenchmarkUnion(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Union(data1, data2)
	}
}

// BenchmarkSort 基准测试：排序性能
func BenchmarkSort(b *testing.B) {
	data := makeRange(0, 1000)
	// 简单反转以给排序增加工作量
	for i := 0; i < len(data)/2; i++ {
		data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OrderBy(From(data), func(i int) int { return i }).ToSlice()
	}
}
