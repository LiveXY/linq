# High-performance generic LINQ in Go

使用方法:
```
go get github.com/LiveXY/linq
```

测试代码:
```
package test

import (
	"fmt"
	"testing"

	"github.com/livexy/linq"
)

type BMember struct {
	Name string
	ID   int64
	Age  int
	Sex  int8
}
type SMember struct {
	Name string
	ID   int64
}

var members = []*BMember{
	{ID: 1, Name: "张三", Sex: 1, Age: 28},
	{ID: 2, Name: "李四", Sex: 2, Age: 28},
	{ID: 3, Name: "王五", Sex: 1, Age: 29},
	{ID: 4, Name: "老六", Sex: 2, Age: 29},
}

func TestSum(t *testing.T) {
	fmt.Printf("Sum Age: %+v \n", linq.From(members).SumIntBy(func(m *BMember) int { return m.Age }))
	fmt.Printf("Avg Age: %+v \n", linq.From(members).AvgIntBy(func(m *BMember) int { return m.Age }))
	fmt.Printf("Sum Age: %+v \n", linq.SumBy(linq.From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("Min Age: %+v \n", linq.MinBy(linq.From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("Max Age: %+v \n", linq.MaxBy(linq.From(members), func(m *BMember) int { return m.Age }))
}
func TestPage(t *testing.T) {
	page, pageSize := 1, 3
	out1 := linq.From(members).Skip((page - 1) * pageSize).Take(pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
	page = 2
	out1 = linq.From(members).Page(page, pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
}
func TestUnion(t *testing.T) {
	out := linq.From(members).Union(linq.From(members)).ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
}

func TestOrder(t *testing.T) {
	query := linq.From(members)
	query = linq.OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	query = linq.ThenBy(query, func(m *BMember) int { return m.Age })
	out4 := query.ToSlice()
	for _, v := range out4 {
		fmt.Printf("%+v \n", v)
	}
}

func TestFrom(t *testing.T) {
	out := linq.From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 }).
		ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
	out2 := linq.Select(
		linq.From(out),
		func(m *BMember) *SMember { return &SMember{ID: m.ID, Name: m.Name} },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
	out3 := linq.GroupBy(
		linq.From(members),
		func(m *BMember) int8 { return m.Sex },
	).ToSlice()
	for _, v := range out3 {
		fmt.Printf("%+v \n", v)
	}
	out4 := linq.GroupBySelect(
		linq.From(members),
		func(m *BMember) int8 { return m.Sex },
		func(m *BMember) *BMember { return m },
	).ToSlice()
	for _, v := range out4 {
		fmt.Printf("%+v \n", v)
	}
}

func TestFilter(t *testing.T) {
	out2 := linq.Filter(
		linq.From(members),
		func(m *BMember) (*SMember, bool) { return nil, false },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
}
func TestHasOrder(t *testing.T) {
	query := linq.From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 })
	fmt.Printf("%+v \n", query.HasOrder())
	query = linq.OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	fmt.Printf("%+v \n", query.HasOrder())
}

func TestFirst(t *testing.T) {
	fmt.Println(1, linq.From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).DefaultIfEmpty(&BMember{}).First())
	fmt.Println(2, linq.From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).First())
}

```

## 性能测试 (Performance Benchmark)

基于 Apple M4 Pro (macOS/arm64) 的测试结果：

| 测试场景 (Benchmark)   | 单次耗时 (ns/op) | 内存 (B/op) | 分配次数 (allocs/op) | 说明 |
|-----------------------|-----------------|-------------|---------------------|------|
| `FromString`          | **7,757**       | **56**      | **2**               | **零拷贝**字符串遍历，内存开销极低 |
| `MinBy`               | 16,396          | 72          | 2                   | 流式处理，极低内存占用 |
| `Where` (Filter)      | 26,303          | 128,352     | 19                  | 10,000 元素过滤 |
| `Union`               | 38,573          | 90,648      | 21                  | 集合合并优化 |
| `FromSlice`           | 45,833          | 357,697     | 21                  | 10,000 元素切片转换 |
| `Select` (Map)        | 45,879          | 357,729     | 22                  | 10,000 元素映射 |
| `Sort`                | 10,760          | 50,712      | 32                  | 1,000 元素排序 |
| `GroupBy`             | 146,036         | 224,864     | 831                 | 10,000 元素确定性分组 |

> **Highlight**: `FromString` 采用了 UTF-8 解码优化，避免了全量 `rune` 数组转换，性能与内存表现卓越。

测试命令: `go test -bench=. -benchmem`

## 高并发场景优化 (High Concurrency Optimization)

本库针对高并发场景进行了深度优化，提供以下特性：

### 🚀 核心特性

#### 1. BufferPool - 切片复用，降低 GC 压力
```go
pool := linq.NewBufferPool[int]()

// 获取复用的 buffer
buf := pool.Get(1000)
result := linq.From(data).Where(filter).AppendTo(buf)

// 使用完后归还
defer pool.Put(result[:0])
```

#### 2. Comparable 类型优化 - 避免装箱，性能提升 42%
```go
// ✅ 推荐：使用优化版本
result := linq.DistinctComparable(linq.From(numbers)).ToSlice()

// ❌ 避免：会产生装箱开销
result := linq.From(numbers).Distinct().ToSlice()
```

**性能对比**（10,000 元素）：
- `DistinctComparable`: 68,812 ns/op, 99,768 B/op, 37 allocs/op
- `Distinct`: 119,023 ns/op, 140,280 B/op, 781 allocs/op
- **提升**: 42% 更快，分配次数减少 95%

#### 3. 并发处理 - 内置 Panic 恢复
```go
// ForEachParallel - 并发执行，自动恢复 panic
linq.From(items).ForEachParallel(10, func(item Item) {
    processItem(item) // 即使 panic 也不会影响其他 worker
})

// SelectAsync - 并发转换，支持提前退出
result := linq.SelectAsync(query, 5, expensiveTransform).
    Take(100).
    ToSlice()
```

### ⚠️ 重要说明

- **Goroutine 安全**: 所有并发方法都已修复 goroutine 泄漏问题
- **Panic 隔离**: `ForEachParallel` 和 `SelectAsync` 内置 panic 恢复机制
- **内存优化**: 使用 `BufferPool` 可降低 60% 的 GC 压力

详细优化报告请查看 [CONCURRENT_OPTIMIZATION.md](./CONCURRENT_OPTIMIZATION.md)
