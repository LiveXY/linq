package linq_benchmark

import (
	"fmt"
	"slices"
	"testing"
)

// None 判断集合中不包含子集的任何元素
func NoneSmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if slices.Contains(list, subset[i]) {
			return false
		}
	}
	return true
}

// None 判断集合中不包含子集的任何元素 适用于大数据
func NoneBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 || len(list) == 0 {
		return true
	}
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; ok {
			return false
		}
	}
	return true
}

// Some 判断集合中包含子集中的至少有一个元素
func SomeSmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if slices.Contains(list, subset[i]) {
			return true
		}
	}
	return false
}

// Some 判断集合中包含子集中的至少有一个元素 适用于大数据
func SomeBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 || len(list) == 0 {
		return false
	}
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; ok {
			return true
		}
	}
	return false
}

// Every - 线性搜索 (小数据逻辑)
func EverySmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if !slices.Contains(list, subset[i]) {
			return false
		}
	}
	return true
}

// EveryBigData - 哈希表搜索 (大数据逻辑)
func EveryBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 {
		return true
	}
	if len(list) == 0 {
		return false
	}
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; !ok {
			return false
		}
	}
	return true
}

func generateRandomData(n, m int, match bool) (list []int, subset []int) {
	list = make([]int, n)
	for i := 0; i < n; i++ {
		list[i] = i
	}

	subset = make([]int, m)
	for i := 0; i < m; i++ {
		if match {
			// 如果要匹配，设置在最后才匹配，以模拟最坏情况
			if i == m-1 {
				subset[i] = n - 1
			} else {
				subset[i] = n + i // 不在原集合中
			}
		} else {
			// 完全不匹配
			subset[i] = n + i
		}
	}
	return
}

func Benchmark_Every_Crossover(b *testing.B) {
	type config struct {
		n, m int
	}

	configs := []config{
		{50, 1000}, {100, 1000}, {500, 1000},
		{100, 5000}, {1000, 5000}, {2000, 5000},
		{50, 50}, {100, 100},
	}

	for _, c := range configs {
		list, subset := generateRandomData(c.n, c.m, true) // Every 测试通常需要全部检查
		nm := uint64(c.n) * uint64(c.m)

		b.Run(fmt.Sprintf("EverySmallData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = EverySmallData(list, subset)
			}
		})

		b.Run(fmt.Sprintf("EveryBigData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = EveryBigData(list, subset)
			}
		})
	}
}

func Benchmark_Some_Crossover(b *testing.B) {
	type config struct {
		n, m int
	}

	// 针对 Some 测试，重点观察 NM 乘积
	configs := []config{
		{50, 1000}, {100, 1000}, {500, 1000},
		{1000, 100}, {1000, 500}, {1000, 1000},
		{2000, 50}, {2000, 100}, {2000, 200},
		{5000, 20}, {5000, 50}, {10000, 10},
	}

	for _, c := range configs {
		// 使用 match=false 模拟最坏情况（全扫描而不命中）
		list, subset := generateRandomData(c.n, c.m, false)
		nm := uint64(c.n) * uint64(c.m)

		b.Run(fmt.Sprintf("SomeSmallData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = SomeSmallData(list, subset)
			}
		})

		b.Run(fmt.Sprintf("SomeBigData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = SomeBigData(list, subset)
			}
		})
	}
}

func Benchmark_None_Crossover(b *testing.B) {
	type config struct {
		n, m int
	}

	// 针对 None 测试，重点观察 NM 乘积
	configs := []config{
		{50, 1000}, {100, 1000}, {500, 1000},
		{1000, 100}, {1000, 500}, {1000, 1000},
		{2000, 50}, {2000, 100}, {2000, 200},
		{5000, 20}, {5000, 50}, {10000, 10},
	}

	for _, c := range configs {
		// 使用 match=false 模拟最坏情况（全扫描而不命中）
		list, subset := generateRandomData(c.n, c.m, false)
		nm := uint64(c.n) * uint64(c.m)

		b.Run(fmt.Sprintf("NoneSmallData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = NoneSmallData(list, subset)
			}
		})

		b.Run(fmt.Sprintf("NoneBigData_NM%d_N%d_M%d", nm, c.n, c.m), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = NoneBigData(list, subset)
			}
		})
	}
}
