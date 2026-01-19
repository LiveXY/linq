package linq_benchmark

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	ahmetb "github.com/ahmetb/go-linq/v3"
	livexy "github.com/livexy/linq"
	lo "github.com/samber/lo"
)

// 数据准备及全局常量
const (
	size = 100000 // 测试数据量：10万条
)

var (
	intData       []int
	duplicateData []int // 包含重复项的数据
	userList      []User
)

// User 定义测试用的结构体
type User struct {
	ID   int
	Name string
	Age  int
}

// 初始化测试数据，包括整数序列、重复数据和结构体切片
func init() {
	intData = make([]int, size)
	for i := 0; i < size; i++ {
		intData[i] = i
	}

	duplicateData = make([]int, size)
	for i := 0; i < size; i++ {
		duplicateData[i] = i % 1000 // 重复出现 0-999（1000个唯一项，重复100次）
	}

	userList = make([]User, size)
	for i := 0; i < size; i++ {
		userList[i] = User{
			ID:   i,
			Name: fmt.Sprintf("用户%d", i),
			Age:  rand.Intn(100),
		}
	}
}

// --- 基准测试: Where (过滤) ---

// Benchmark_LiveXY_Where 测试 LiveXY 库的过滤性能
func Benchmark_LiveXY_Where(b *testing.B) {
	var query = livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = query.Where(func(i int) bool {
			return i%2 == 0
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Where 测试 go-linq (ahmetb) 库的过滤性能
func Benchmark_Ahmetb_Where(b *testing.B) {
	var query = ahmetb.From(intData)
	for i := 0; i < b.N; i++ {
		var res []int
		query.Where(func(i interface{}) bool {
			return i.(int)%2 == 0
		}).ToSlice(&res)
	}
}

// Benchmark_Lo_Where 测试 lo 库的过滤性能
func Benchmark_Lo_Where(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Filter(intData, func(i int, _ int) bool {
			return i%2 == 0
		})
	}
}

// Benchmark_Native_Where 测试原生 Go for 循环的过滤性能
func Benchmark_Native_Where(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var res []int
		// 注意：为了公平对比，这里不预分配容量。
		// 实际上大多数库在执行 Where 时也无法预知结果集大小。
		for _, v := range intData {
			if v%2 == 0 {
				res = append(res, v)
			}
		}
	}
}

// --- 基准测试: Select (映射) ---

// Benchmark_LiveXY_Select 测试 LiveXY 库的映射性能
func Benchmark_LiveXY_Select(b *testing.B) {
	q := livexy.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Select(q, func(i int) int {
			return i * 2
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Select 测试 go-linq (ahmetb) 库的映射性能
func Benchmark_Ahmetb_Select(b *testing.B) {
	query := ahmetb.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		query.Select(func(i interface{}) interface{} {
			return i.(int) * 2
		}).ToSlice(&res)
	}
}

// Benchmark_Lo_Select 测试 lo 库的映射性能
func Benchmark_Lo_Select(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Map(intData, func(i int, _ int) int {
			return i * 2
		})
	}
}

// Benchmark_Native_Select 测试原生 Go for 循环的映射性能
func Benchmark_Native_Select(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		for i, v := range intData {
			res[i] = v * 2
		}
	}
}

// --- 基准测试: 链式调用 (Where + Select) ---

// Benchmark_LiveXY_Chain 测试 LiveXY 库的链式调用 (过滤+映射) 性能
func Benchmark_LiveXY_Chain(b *testing.B) {
	query := livexy.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := query.Where(func(i int) bool {
			return i%2 == 0
		})
		_ = livexy.Select(q, func(i int) int {
			return i * 2
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Chain 测试 go-linq (ahmetb) 库的链式调用性能
func Benchmark_Ahmetb_Chain(b *testing.B) {
	query := ahmetb.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		query.Where(func(i interface{}) bool {
			return i.(int)%2 == 0
		}).Select(func(i interface{}) interface{} {
			return i.(int) * 2
		}).ToSlice(&res)
	}
}

// Benchmark_Lo_Chain 测试 lo 库的链式调用性能
func Benchmark_Lo_Chain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// lo 是及早求值的（Eager），会创建中间临时切片
		filtered := lo.Filter(intData, func(i int, _ int) bool {
			return i%2 == 0
		})
		_ = lo.Map(filtered, func(i int, _ int) int {
			return i * 2
		})
	}
}

// Benchmark_Native_Chain 测试原生 Go 实现的链式处理性能
func Benchmark_Native_Chain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var res []int
		for _, v := range intData {
			if v%2 == 0 {
				res = append(res, v*2)
			}
		}
	}
}

// --- 基准测试: 结构体处理 (过滤年龄 > 18, 映射出姓名) ---

// Benchmark_LiveXY_Struct 测试 LiveXY 库处理结构体切片的性能
func Benchmark_LiveXY_Struct(b *testing.B) {
	query := livexy.From(userList)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := query.Where(func(u User) bool {
			return u.Age > 18
		})
		livexy.Select(q, func(u User) string {
			return u.Name
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Struct 测试 go-linq (ahmetb) 库处理结构体切片的性能
func Benchmark_Ahmetb_Struct(b *testing.B) {
	query := ahmetb.From(userList)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []string
		query.Where(func(i interface{}) bool {
			return i.(User).Age > 18
		}).Select(func(i interface{}) interface{} {
			return i.(User).Name
		}).ToSlice(&res)
	}
}

// Benchmark_Lo_Struct 测试 lo 库处理结构体切片的性能
func Benchmark_Lo_Struct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := lo.Filter(userList, func(u User, _ int) bool {
			return u.Age > 18
		})
		lo.Map(filtered, func(u User, _ int) string {
			return u.Name
		})
	}
}

// Benchmark_Native_Struct 测试原生 Go 实现处理结构体的性能
func Benchmark_Native_Struct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var res []string
		for _, u := range userList {
			if u.Age > 18 {
				res = append(res, u.Name)
			}
		}
	}
}

// --- 基准测试: 结构体排序 (OrderBy) ---

// Benchmark_LiveXY_Sort 测试 LiveXY 库的排序性能
func Benchmark_LiveXY_Sort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		livexy.OrderBy(q, func(u User) int {
			return u.Age
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Sort 测试 go-linq (ahmetb) 库的排序性能
func Benchmark_Ahmetb_Sort(b *testing.B) {
	smallData := userList[:1000]
	query := ahmetb.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []User
		query.OrderBy(func(i interface{}) interface{} {
			return i.(User).Age
		}).ToSlice(&res)
	}
}

// Benchmark_Native_Sort 测试原生 Go sort.Slice 的排序性能
func Benchmark_Native_Sort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		sort.Slice(smallData, func(i, j int) bool { return smallData[i].Age < smallData[j].Age })
	}
}

// --- 基准测试: 去重 (Distinct) ---

// Benchmark_LiveXY_Distinct 测试 LiveXY 库的去重性能
func Benchmark_LiveXY_Distinct(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = query.Distinct().ToSlice()
	}
}

// Benchmark_Linq1_Distinct 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_Linq1_Distinct(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Distinct(query, func(i int) int {
			return i
		}).ToSlice()
	}
}

// Benchmark_Ahmetb_Distinct 测试 go-linq (ahmetb) 库的去重性能
func Benchmark_Ahmetb_Distinct(b *testing.B) {
	query := ahmetb.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		query.Distinct().ToSlice(&res)
	}
}

// Benchmark_Lo_Distinct 测试 lo 库的去重性能
func Benchmark_Lo_Distinct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Uniq(duplicateData)
	}
}

// Benchmark_Native_Distinct 测试使用 map 实现的原生去重性能
func Benchmark_Native_Distinct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{})
		var res []int
		for _, v := range duplicateData {
			if _, ok := set[v]; !ok {
				set[v] = struct{}{}
				res = append(res, v)
			}
		}
	}
}
