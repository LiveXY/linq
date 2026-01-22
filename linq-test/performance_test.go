package linq_benchmark

import (
	"fmt"
	"math/rand"
	"slices"
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
	intDataOther  []int // 用于 Union 测试的另一组数据
	intSubset     []int // 用于 Every 测试的子集
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

	intDataOther = make([]int, size)
	for i := 0; i < size; i++ {
		intDataOther[i] = i + size/2 // 与 intData 有一半重叠
	}

	intSubset = make([]int, size/10)
	for i := 0; i < size/10; i++ {
		intSubset[i] = rand.Intn(size) // 随机在 0 到 size-1 之间取值
	}

	fmt.Println(len(intData), len(intSubset))

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

// Benchmark_LiveXY2_Where 测试 LiveXY 库的过滤性能
func Benchmark_LiveXY2_Where(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = livexy.Where(intData, func(i int) bool {
			return i%2 == 0
		})
	}
}

// Benchmark_LiveXY3_Where 测试 LiveXY 库的过滤性能
func Benchmark_LiveXY3_Where(b *testing.B) {
	var query = livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = livexy.WhereSelect(query, func(i int) (int, bool) {
			return i, i%2 == 0
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

// Benchmark_LiveXY2_Select 测试 LiveXY 库的映射性能
func Benchmark_LiveXY2_Select(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = livexy.Map(intData, func(i int) int {
			return i * 2
		})
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

// Benchmark_LiveXY2_Chain 测试 LiveXY 库的链式调用 (过滤+映射) 性能
func Benchmark_LiveXY2_Chain(b *testing.B) {
	query := livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = livexy.WhereSelect(query, func(i int) (int, bool) {
			return i * 2, i%2 == 0
		}).ToSlice()
	}
}

// Benchmark_LiveXY3_Chain 测试 LiveXY 库的链式调用性能
func Benchmark_LiveXY3_Chain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := livexy.Where(intData, func(i int) bool {
			return i%2 == 0
		})
		_ = livexy.Map(filtered, func(i int) int {
			return i * 2
		})
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

// Benchmark_LiveXY2_Struct 测试 LiveXY 库处理结构体切片的性能
func Benchmark_LiveXY2_Struct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := livexy.Where(userList, func(u User) bool {
			return u.Age > 18
		})
		livexy.Map(filtered, func(u User) string {
			return u.Name
		})
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

// Benchmark_Slices_Sort 测试原生 Go slices.SortFunc 的排序性能 (Go 1.21+)
func Benchmark_Slices_Sort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortFunc(smallData, func(a, b User) int {
			return a.Age - b.Age
		})
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

// Benchmark_LiveXY2_Distinct 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_LiveXY2_Distinct(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.DistinctSelect(query, func(i int) int {
			return i
		}).ToSlice()
	}
}

// Benchmark_Uniq_Distinct 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_Uniq_Distinct(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Uniq(duplicateData)
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

// --- 基准测试: 并集 (Union) ---

// Benchmark_LiveXY_Union 测试 LiveXY 库的并集去重性能
func Benchmark_LiveXY_Union(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q1.Union(q2).ToSlice()
	}
}

// Benchmark_LiveXY2_Union 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_LiveXY2_Union(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.UnionSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// Benchmark_LiveXY3_Union 测试使用 LiveXY.Intersect 的交集性能
func Benchmark_LiveXY3_Union(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Union(intData, intDataOther)
	}
}

// Benchmark_Ahmetb_Union 测试 go-linq (ahmetb) 库的并集去重性能
func Benchmark_Ahmetb_Union(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Union(q2).ToSlice(&res)
	}
}

// Benchmark_Lo_Union 测试 lo 库的并集去重性能
func Benchmark_Lo_Union(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Union(intData, intDataOther)
	}
}

// Benchmark_Native_Union 测试使用 map 实现的原生并集去重性能
func Benchmark_Native_Union(b *testing.B) {
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, size)
		var res []int
		for _, v := range intData {
			if _, ok := set[v]; !ok {
				set[v] = struct{}{}
				res = append(res, v)
			}
		}
		for _, v := range intDataOther {
			if _, ok := set[v]; !ok {
				set[v] = struct{}{}
				res = append(res, v)
			}
		}
	}
}

// --- 基准测试: 包含 (Contains) ---

// Benchmark_LiveXY_Contains 测试 LiveXY 库的包含查询性能 (查找末尾元素)
func Benchmark_LiveXY_Contains(b *testing.B) {
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Contains(intData, target)
	}
}

// Benchmark_Ahmetb_Contains 测试 go-linq (ahmetb) 库的包含查询性能
func Benchmark_Ahmetb_Contains(b *testing.B) {
	q := ahmetb.From(intData)
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Contains(target)
	}
}

// Benchmark_Lo_Contains 测试 lo 库的包含查询性能
func Benchmark_Lo_Contains(b *testing.B) {
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Contains(intData, target)
	}
}

// Benchmark_Native_Contains 测试原生 Go for 循环的包含查询性能
func Benchmark_Native_Contains(b *testing.B) {
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		found := false
		for _, v := range intData {
			if v == target {
				found = true
				break
			}
		}
		_ = found
	}
}

// --- 基准测试: 是否包含全部子集 (Every) ---

// Benchmark_LiveXY_Every 测试 LiveXY 库的 Every 性能
func Benchmark_LiveXY_Every(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Every(intData, intSubset)
	}
}

// Benchmark_Ahmetb_Every 测试 go-linq (ahmetb) 库的 Every 性能 (组合实现)
func Benchmark_Ahmetb_Every(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ahmetb.From(intSubset).All(func(i interface{}) bool {
			return ahmetb.From(intData).Contains(i)
		})
	}
}

// Benchmark_Lo_Every 测试 lo 库的 Every 性能
func Benchmark_Lo_Every(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Every(intData, intSubset)
	}
}

// Benchmark_Native_Every 测试原生 Go 实现的 Every 性能
func Benchmark_Native_Every(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, len(intData))
		for _, v := range intData {
			set[v] = struct{}{}
		}
		all := true
		for _, v := range intSubset {
			if _, ok := set[v]; !ok {
				all = false
				break
			}
		}
		_ = all
	}
}

// --- 基准测试: 是否包含子集中的任意元素 (Some) ---

// Benchmark_LiveXY_Some 测试 LiveXY 库的 Some 性能
func Benchmark_LiveXY_Some(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Some(intData, intSubset)
	}
}

// Benchmark_Ahmetb_Some 测试 go-linq (ahmetb) 库的 Some 性能 (组合实现)
func Benchmark_Ahmetb_Some(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ahmetb.From(intSubset).AnyWith(func(i interface{}) bool {
			return ahmetb.From(intData).Contains(i)
		})
	}
}

// Benchmark_Lo_Some 测试 lo 库的 Some 性能
func Benchmark_Lo_Some(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Some(intData, intSubset)
	}
}

// Benchmark_Native_Some 测试原生 Go 实现的 Some 性能
func Benchmark_Native_Some(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, len(intData))
		for _, v := range intData {
			set[v] = struct{}{}
		}
		any := false
		for _, v := range intSubset {
			if _, ok := set[v]; ok {
				any = true
				break
			}
		}
		_ = any
	}
}

// --- 基准测试: 是否都不包含 (None) ---

// Benchmark_LiveXY_None 测试 LiveXY 库的 None 性能
// None 的逻辑是：集合 A 中没有任何元素属于集合 B。
// 等价于 !Some(A, B)
func Benchmark_LiveXY_None(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.None(intData, intSubset)
	}
}

// Benchmark_Ahmetb_None 测试 go-linq (ahmetb) 库的 None 性能 (组合实现)
func Benchmark_Ahmetb_None(b *testing.B) {
	b.ResetTimer()
	var q = ahmetb.From(intSubset)
	for i := 0; i < b.N; i++ {
		// All 返回 true 如果 predicate 对所有元素都为 true
		// 这里 predicate 是 "不包含"，所以 All(不包含) == None(包含)
		_ = q.All(func(i interface{}) bool {
			return !ahmetb.From(intData).Contains(i)
		})
	}
}

// Benchmark_Lo_None 测试 lo 库的 None 性能
func Benchmark_Lo_None(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.None(intData, intSubset)
	}
}

// Benchmark_Native_None 测试原生 Go 实现的 None 性能
func Benchmark_Native_None(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, len(intData))
		for _, v := range intData {
			set[v] = struct{}{}
		}
		none := true
		for _, v := range intSubset {
			if _, ok := set[v]; ok {
				none = false
				break
			}
		}
		_ = none
	}
}

// --- 基准测试: 合并 (Concat) ---

// Benchmark_LiveXY_Concat 测试 LiveXY 库的合并性能
func Benchmark_LiveXY_Concat(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q1.Concat(q2).ToSlice()
	}
}

// Benchmark_LiveXY2_Concat 测试 LiveXY 库的合并性能
func Benchmark_LiveXY2_Concat(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Concat(intData, intDataOther)
	}
}

// Benchmark_Ahmetb_Concat 测试 go-linq (ahmetb) 库的合并性能
func Benchmark_Ahmetb_Concat(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Concat(q2).ToSlice(&res)
	}
}

// Benchmark_Lo_Concat 测试 lo 库的合并性能
func Benchmark_Lo_Concat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Flatten([][]int{intData, intDataOther})
	}
}

// Benchmark_Native_Concat 测试原生 Go append 的合并性能
func Benchmark_Native_Concat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, 0, len(intData)+len(intDataOther))
		res = append(res, intData...)
		res = append(res, intDataOther...)
		_ = res
	}
}

// --- 基准测试: 交集 (Intersect) ---

// Benchmark_LiveXY_Intersect 测试 LiveXY 库的交集性能
func Benchmark_LiveXY_Intersect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q1.Intersect(q2).ToSlice()
	}
}

// Benchmark_LiveXY2_Intersect 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_LiveXY2_Intersect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.IntersectSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// Benchmark_LiveXY3_Intersect 测试使用 LiveXY.Intersect 的交集性能
func Benchmark_LiveXY3_Intersect(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Intersect(intData, intDataOther)
	}
}

// Benchmark_Ahmetb_Intersect 测试 go-linq (ahmetb) 库的交集性能
func Benchmark_Ahmetb_Intersect(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Intersect(q2).ToSlice(&res)
	}
}

// Benchmark_Lo_Intersect 测试 lo 库的交集性能
func Benchmark_Lo_Intersect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Intersect(intData, intDataOther)
	}
}

// Benchmark_Native_Intersect 测试原生 Go 使用 map 的交集性能
func Benchmark_Native_Intersect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, len(intData))
		for _, v := range intData {
			set[v] = struct{}{}
		}
		var res []int
		for _, v := range intDataOther {
			if _, ok := set[v]; ok {
				res = append(res, v)
			}
		}
		_ = res
	}
}

// --- 基准测试: 差集 (Except) ---

// Benchmark_LiveXY_Except 测试 LiveXY 库的差集性能
func Benchmark_LiveXY_Except(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q1.Except(q2).ToSlice()
	}
}

// Benchmark_LiveXY2_Except 测试使用 LiveXY.Distinct 的自定义键去重性能
func Benchmark_LiveXY2_Except(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.ExceptSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// Benchmark_LiveXY3_Except 测试使用 LiveXY.Intersect 的交集性能
func Benchmark_LiveXY3_Except(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = livexy.Difference(intData, intDataOther)
	}
}

// Benchmark_Ahmetb_Except 测试 go-linq (ahmetb) 库的差集性能
func Benchmark_Ahmetb_Except(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Except(q2).ToSlice(&res)
	}
}

// Benchmark_Lo_Except 测试 lo 库的差集性能 (Difference 只取左差集)
func Benchmark_Lo_Except(b *testing.B) {
	for i := 0; i < b.N; i++ {
		left, _ := lo.Difference(intData, intDataOther)
		_ = left
	}
}

// Benchmark_Native_Except 测试原生 Go 实现的差集性能
func Benchmark_Native_Except(b *testing.B) {
	for i := 0; i < b.N; i++ {
		set := make(map[int]struct{}, len(intDataOther))
		for _, v := range intDataOther {
			set[v] = struct{}{}
		}
		var res []int
		for _, v := range intData {
			if _, ok := set[v]; !ok {
				res = append(res, v)
			}
		}
		_ = res
	}
}

// --- 基准测试: 反转 (Reverse) ---

// Benchmark_LiveXY_Reverse 测试 LiveXY 库的链式反转性能
func Benchmark_LiveXY_Reverse(b *testing.B) {
	q := livexy.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Reverse().ToSlice()
	}
}

// Benchmark_LiveXY2_Reverse 测试 LiveXY 库的静态反转性能
func Benchmark_LiveXY2_Reverse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Reverse(intData)
	}
}

// Benchmark_LiveXY3_Reverse 测试 LiveXY 库的静态反转性能
func Benchmark_LiveXY3_Reverse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.CloneReverse(intData)
	}
}

// Benchmark_Ahmetb_Reverse 测试 go-linq (ahmetb) 库的反转性能
func Benchmark_Ahmetb_Reverse(b *testing.B) {
	q := ahmetb.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q.Reverse().ToSlice(&res)
	}
}

// Benchmark_Lo_Reverse 测试 lo 库的反转性能
func Benchmark_Lo_Reverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Reverse(intData)
	}
}

// Benchmark_Native_Reverse 测试原生 Go 实现的反转性能
func Benchmark_Native_Reverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		n := len(intData)
		for j := 0; j < n; j++ {
			res[j] = intData[n-1-j]
		}
		_ = res
	}
}

// Benchmark_Native_Inplace_Reverse 测试原生 Go 原地反转性能
func Benchmark_Native_Inplace_Reverse(b *testing.B) {
	data := make([]int, len(intData))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(data, intData) // 每次为了测试原地性能，必须先还原数据
		n := len(data)
		for j := 0; j < n/2; j++ {
			data[j], data[n-1-j] = data[n-1-j], data[j]
		}
	}
}

// Benchmark_Lo_Clone_Reverse 测试 lo 库带拷贝的反转性能 (为了公平对比)
func Benchmark_Lo_Clone_Reverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := make([]int, len(intData))
		copy(data, intData)
		_ = lo.Reverse(data)
	}
}

// --- 基准测试: 随机洗牌 (Shuffle) ---

// Benchmark_LiveXY_Shuffle 测试 LiveXY 库的随机洗牌性能 (含拷贝)
func Benchmark_LiveXY_Shuffle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Shuffle(intData)
	}
}

// Benchmark_Lo_Shuffle 测试 lo 库的随机洗牌性能 (原地修改)
func Benchmark_Lo_Shuffle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Shuffle(intData)
	}
}

// Benchmark_Native_Shuffle 测试原生 Go 实现的随机洗牌性能 (含拷贝)
func Benchmark_Native_Shuffle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		copy(res, intData)
		rand.Shuffle(len(res), func(i, j int) {
			res[i], res[j] = res[j], res[i]
		})
		_ = res
	}
}

// --- 实验性: 优化版 Some ---

// SomeOptimized 是尝试引入启发式算法的 Some 实现
func SomeOptimized[T comparable](collection []T, subset []T) bool {
	n1 := len(collection)
	n2 := len(subset)

	if n1 == 0 || n2 == 0 {
		return false
	}

	// 1. 小数据量直接暴力 (Threshold = 128)
	if n1 < 128 || n2 < 128 {
		for _, v := range collection {
			for _, s := range subset {
				if v == s {
					return true
				}
			}
		}
		return false
	}

	// 2. 启发式：尝试快速命中
	// 检查 collection 的前 K 个元素是否在 subset 中。
	const speculationLimit = 50
	limit := speculationLimit
	if n1 < limit {
		limit = n1
	}

	// 提前进行少量双重循环扫描，期望在高命中率场景下快速返回
	for i := 0; i < limit; i++ {
		v := collection[i]
		for _, s := range subset {
			if v == s {
				return true
			}
		}
	}

	// 3. 回退策略：构建 Map (Set)
	// 总是对较小的集合构建 Map
	if n1 < n2 {
		set := make(map[T]struct{}, n1)
		for _, v := range collection {
			set[v] = struct{}{}
		}
		for _, v := range subset {
			if _, ok := set[v]; ok {
				return true
			}
		}
	} else {
		set := make(map[T]struct{}, n2)
		for _, v := range subset {
			set[v] = struct{}{}
		}
		// 跳过已检查的部分
		start := speculationLimit
		if start > n1 {
			start = n1
		}
		for i := start; i < n1; i++ {
			if _, ok := set[collection[i]]; ok {
				return true
			}
		}
	}

	return false
}

// Benchmark_LiveXY_Optimized_Some 测试 LiveXY 库的 Some 性能 (优化版)
func Benchmark_LiveXY_Optimized_Some(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SomeOptimized(intData, intSubset)
	}
}
