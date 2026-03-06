// go test -bench=Benchmark -benchmem linq_benchmark_test.go linq.go

package linq

import (
	"context"
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
	var query = From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Where(func(i int) bool { return i%2 == 0 }).ToSlice()
	}
}

// BenchmarkSelect 基准测试：映射操作
func BenchmarkSelect(b *testing.B) {
	data := makeRange(0, 10000)
	var query = From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Select(query, func(i int) int { return i * 2 }).ToSlice()
	}
}

// BenchmarkMinBy 基准测试：按条件查找最小值
func BenchmarkMinBy(b *testing.B) {
	data := makeRange(0, 10000)
	var query = From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MinBy(query, func(i int) int { return i })
	}
}

// BenchmarkGroupBy 基准测试：分组操作
func BenchmarkGroupBy(b *testing.B) {
	data := makeRange(0, 10000)
	var query = From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupBy(query, func(i int) int { return i % 100 }).ToSlice()
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
func BenchmarkSliceUnion(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceUnion(data1, data2)
	}
}

// BenchmarkSort 基准测试：排序性能
func BenchmarkSort(b *testing.B) {
	data := makeRange(0, 1000)
	var query = From(data)
	for i := 0; i < len(data)/2; i++ {
		data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.HasOrder()
		OrderByDescending(query, func(i int) int { return i }).ToSlice()
		ThenBy(OrderBy(query, func(i int) int { return i }), func(i int) int { return i }).ToSlice()
		ThenByDescending(OrderBy(query, func(i int) int { return i }), func(i int) int { return i }).ToSlice()
	}
}

// BenchmarkFromMap 基准测试：从 Map 创建查询
func BenchmarkFromMap(b *testing.B) {
	data := make(map[int]int)
	for i := 0; i < 1000; i++ {
		data[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromMap(data).ToSlice()
	}
}

// BenchmarkQueryRange 基准测试：数值范围生成
func BenchmarkQueryRange(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		QueryRange(0, 1000).ToSlice()
	}
}

// BenchmarkDistinct 基准测试：去重操作
func BenchmarkDistinct(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i % 10
	}
	var query = From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Distinct(query).ToSlice()
	}
}

// BenchmarkIntersect 基准测试：交集操作
func BenchmarkIntersect(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	q1 := From(data1)
	q2 := From(data2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Intersect(q1, q2).ToSlice()
	}
}

// BenchmarkExcept 基准测试：差集操作
func BenchmarkExcept(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	q1 := From(data1)
	q2 := From(data2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Except(q1, q2).ToSlice()
	}
}

// BenchmarkConcat 基准测试：连接操作
func BenchmarkConcat(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(1000, 2000)
	q1 := From(data1)
	q2 := From(data2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q1.Concat(q2).ToSlice()
	}
}

// BenchmarkSelectAsync 基准测试：并发映射
func BenchmarkSelectAsync(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SelectAsync(q, func(i int) int { return i * 2 }, 4).ToSlice()
	}
}

// BenchmarkAllAnyCount 基准测试：终端谓词操作
func BenchmarkAllAnyCount(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.All(func(i int) bool { return i >= 0 })
		q.AnyWith(func(i int) bool { return i > 500 })
		q.CountWith(func(i int) bool { return i%2 == 0 })
	}
}

// BenchmarkFirstLast 基准测试：查找首尾
func BenchmarkFirstLast(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.FirstWith(func(i int) bool { return i > 500 })
		q.LastWith(func(i int) bool { return i < 500 })
	}
}

// BenchmarkSumAverage 基准测试：聚合计算
func BenchmarkSumAverage(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SumBy(q, func(i int) int { return i })
		AverageBy(q, func(i int) float64 { return float64(i) })
		MaxBy(q, func(i int) int { return i })
	}
}

// BenchmarkToMap 基准测试：转为 Map
func BenchmarkToMap(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToMap(q, func(i int) int { return i })
	}
}

// BenchmarkSliceUtilities 基准测试：切片工具函数
func BenchmarkSliceUtilities(b *testing.B) {
	data := makeRange(0, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains(From(data), 500)
		IndexOf(From(data), 500)
		SliceUniq(data)
		SliceReverse(data)
		SliceShuffle(data)
	}
}

// BenchmarkCollectionOps 基准测试：集合工具函数
func BenchmarkCollectionOps(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceEvery(data1, data2[:10])
		SliceSome(data1, data2[:100])
		SliceDifference(data1, data2)
		SliceIntersect(data1, data2)
	}
}

// BenchmarkWithout 基准测试：移除元素
func BenchmarkWithout(b *testing.B) {
	data := makeRange(0, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceWithout(data, 1, 2, 3, 4, 5)
		SliceWithoutIndex(data, 0, 10, 100)
	}
}

// BenchmarkOtherOps 基准测试：其余操作
func BenchmarkOtherOps(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Page(2, 100).ToSlice()
		QueryRepeat(1, 1000).ToSlice()
		WhereSelect(q, func(i int) (int, bool) { return i, i%2 == 0 }).ToSlice()
		q.Single()
		q.Append(1001).ToSlice()
		q.Prepend(-1).ToSlice()
		q.DefaultIfEmpty(0).ToSlice()
		Union(q, From(data)).ToSlice()
		q.Reverse().ToSlice()
	}
}

// BenchmarkTerminalLoop 基准测试：带循环的终端操作
func BenchmarkTerminalLoop(b *testing.B) {
	data := makeRange(0, 100)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Any()
		q.First()
		q.FirstDefault(0)
		q.ForEach(func(i int) bool { return true })
		q.ForEachIndexed(func(idx, val int) bool { return true })
		q.ForEachParallel(func(i int) {}, 2)
		q.IndexOfWith(func(i int) bool { return i == 50 })
	}
}

// BenchmarkWhileOps 基准测试：While 相关操作
func BenchmarkWhileOps(b *testing.B) {
	data := makeRange(0, 100)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.TakeWhile(func(i int) bool { return i < 50 }).ToSlice()
		q.SkipWhile(func(i int) bool { return i < 50 }).ToSlice()
	}
}

// BenchmarkDataSource 基准测试：更多数据源
func BenchmarkDataSource(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := make(chan int, 100)
		for j := 0; j < 100; j++ {
			ch <- j
		}
		close(ch)
		FromChannel(ch).ToSlice()
	}
}

// BenchmarkUtilityFns 基准测试：逻辑工具函数
func BenchmarkUtilityFns(b *testing.B) {
	data := makeRange(0, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Default(0, 1)
		IsEmpty(0)
		IsNotEmpty(1)
		TryDelay(func() error { return nil })
		IF(true, 1, 2)
		Empty[int]()
		SliceRand(data, 10)
		SliceEqual(data, data...)
		SliceEqualBy(data, data, func(i int) int { return i })
	}
}

// BenchmarkAggregates 基准测试：各种类型的聚合
func BenchmarkAggregates(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SumBy(q, func(i int) int8 { return int8(i % 127) })
		SumBy(q, func(i int) int16 { return int16(i % 32767) })
		SumBy(q, func(i int) int32 { return int32(i) })
		SumBy(q, func(i int) int64 { return int64(i) })
		SumBy(q, func(i int) float32 { return float32(i) })
		SumBy(q, func(i int) float64 { return float64(i) })
		SumBy(q, func(i int) uint8 { return uint8(i % 255) })
		SumBy(q, func(i int) uint16 { return uint16(i % 65535) })
		SumBy(q, func(i int) uint32 { return uint32(i) })
		SumBy(q, func(i int) uint64 { return uint64(i) })
		SumBy(q, func(i int) uint { return uint(i) })
		AverageBy(q, func(i int) int { return i })
		AverageBy(q, func(i int) int64 { return int64(i) })
	}
}

// BenchmarkStaticAggregates 基准测试：静态聚合函数
func BenchmarkStaticAggregates(b *testing.B) {
	data := makeRange(0, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sum(From(data))
		SliceMin(data...)
		SliceMax(data...)
	}
}

// BenchmarkAdvancedProjections 基准测试：高级映射与集合操作
func BenchmarkAdvancedProjections(b *testing.B) {
	data1 := makeRange(0, 1000)
	data2 := makeRange(500, 1500)
	q1 := From(data1)
	q2 := From(data2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DistinctSelect(q1, func(i int) int { return i % 10 }).ToSlice()
		UnionSelect(q1, q2, func(i int) int { return i }).ToSlice()
		IntersectSelect(q1, q2, func(i int) int { return i }).ToSlice()
		ExceptSelect(q1, q2, func(i int) int { return i }).ToSlice()
		GroupBySelect(q1, func(i int) int { return i % 10 }, func(i int) int { return i }).ToSlice()
	}
}

// BenchmarkTerminalOutputs 基准测试：各种终端输出
func BenchmarkTerminalOutputs(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dest := make([]int, 0, 1000)
		q.AppendTo(dest)
		q.ToMapSlice(func(i int) map[string]int { return map[string]int{"key": i} })
	}
}

// BenchmarkFilteringUtils 基准测试：过滤工具
func BenchmarkFilteringUtils(b *testing.B) {
	data := makeRange(-500, 500)
	strs := []string{"a", "", "b", "", "c"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceWithoutEmpty(strs)
		SliceWithoutLEZero(data)
	}
}

// BenchmarkMoreUtilityFns 基准测试：更多逻辑工具
func BenchmarkMoreUtilityFns(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TryCatch(func() error { panic("error") }, func() {})
	}
}

// BenchmarkStaticFunctions 基准测试：全局静态工具函数
func BenchmarkStaticFunctions(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceMap(data, func(i int) int { return i * 2 })
		SliceMapIndexed(data, func(i int, idx int) int { return i + idx })
		SliceWhere(data, func(i int) bool { return i > 500 })
		SliceWhereIndexed(data, func(i int, idx int) bool { return idx%2 == 0 })
		SelectAsyncCtx(ctx, q, func(i int) int { return i }, 4)
	}
}

// BenchmarkSearchUtilities 基准测试：更多搜索工具
func BenchmarkSearchUtilities(b *testing.B) {
	data := makeRange(0, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceContainsBy(data, func(i int) bool { return i == 500 })
		LastIndexOf(From(data), 500)
	}
}

// BenchmarkBigDataOps 基准测试：大数据集合操作
func BenchmarkBigDataOps(b *testing.B) {
	data1 := makeRange(0, 3000) // 超过 2000 触发 BigData 优化路径
	data2 := makeRange(2500, 3500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceEvery(data1, data2[:10])
		SliceEvery(data1, data1[1000:2500]) // 触发 EveryBigData
		SliceSome(data1, data2[:100])
		SliceNone(data1, data2[:10])
	}
}

// BenchmarkOutputChannels 基准测试：输出到 Channel
func BenchmarkOutputChannels(b *testing.B) {
	data := makeRange(0, 100)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.ToChannel(context.Background())
	}
}

// BenchmarkTerminalOps 基准测试：更多终端操作
func BenchmarkTerminalOps(b *testing.B) {
	data := makeRange(0, 100)
	q := From(data)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.ForEachParallelCtx(ctx, func(i int) {}, 4)
		q.Last()
		q.LastDefault(0)

		// 针对 Every/Some/None 的不同数据路径
		small := []int{10}
		SliceEvery(data, small)
		SliceSome(data, small)
	}
}

// BenchmarkOrderedQuerySort 基准测试：新版高性能排序 (Single)
func BenchmarkOrderedQuerySort(b *testing.B) {
	data := makeRange(0, 1000)
	// 乱序
	for i := 0; i < len(data)/2; i++ {
		data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
	}
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(Asc(func(i int) int { return i })).ToSlice()
	}
}

// BenchmarkOrderedQueryThen 基准测试：新版高性能多级排序 (Then)
func BenchmarkOrderedQueryThen(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(Asc(func(i int) int { return i })).
			Then(Desc(func(i int) int { return i })).
			ToSlice()
	}
}

// BenchmarkOrderedQueryOperations 基准测试：排序后的操作 (Take, Where)
func BenchmarkOrderedQueryOperations(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Take 触发全量排序后切片 (目前实现)
		q.Order(Asc(func(i int) int { return i })).Take(10).ToSlice()

		// Where 触发全量排序后过滤
		q.Order(Asc(func(i int) int { return i })).Where(func(i int) bool { return i%2 == 0 }).ToSlice()
	}
}

// BenchmarkOrderedQueryFirst 基准测试：排序后取第一个 (O(N) 优化验证)
func BenchmarkOrderedQueryFirst(b *testing.B) {
	data := makeRange(0, 10000) // 较大数据量以突显 O(N) vs O(N log N) 差异
	for i := 0; i < len(data)/2; i++ {
		data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
	}
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(Asc(func(i int) int { return i })).First()
	}
}

// BenchmarkOrderedQueryReverse 基准测试：排序后反转 (Zero-Allocation 验证)
func BenchmarkOrderedQueryReverse(b *testing.B) {
	data := makeRange(0, 1000)
	q := From(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reverse 仅包装比较器，不执行排序，消耗极低
		// 为了触发实际工作，我们调用 ToSlice
		q.Order(Asc(func(i int) int { return i })).Reverse().ToSlice()
	}
}
