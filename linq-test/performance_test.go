// go test -bench=Benchmark -benchmem .
// go test -bench=Sort -benchmem .
package linq_benchmark

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"testing"

	ahmetb "github.com/ahmetb/go-linq/v3"
	livexy "github.com/livexy/linq"
	lo "github.com/samber/lo"
	"github.com/samber/lo/mutable"
)

// 数据准备及全局常量
const (
	size = 100000 // 测试数据量：10万条
	seed = 20260306
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
	ID     int
	Name   string
	Age    int
	Gender int
}

// 初始化测试数据，包括整数序列、重复数据和结构体切片
func init() {
	rng := rand.New(rand.NewPCG(seed, seed+1))

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
		intSubset[i] = rng.IntN(size) // 固定随机种子，保证可复现
	}

	duplicateData = make([]int, size)
	for i := 0; i < size; i++ {
		duplicateData[i] = i % 1000 // 重复出现 0-999（1000个唯一项，重复100次）
	}

	userList = make([]User, size)
	for i := 0; i < size; i++ {
		userList[i] = User{
			ID:     i,
			Name:   fmt.Sprintf("用户%d", i),
			Age:    rng.IntN(100),
			Gender: rng.IntN(2),
		}
	}
}

// --- 基准测试: Where (过滤) ---

// BenchmarkLiveXYWhere 测试 LiveXY 库的过滤性能
func BenchmarkLiveXYWhere(b *testing.B) {
	var query = livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = query.Where(func(i int) bool {
			return i%2 == 0
		}).ToSlice()
	}
}

// BenchmarkLiveXYWhereSlice 测试 LiveXY 库的切片级过滤性能
func BenchmarkLiveXYWhereSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = livexy.Where(intData, func(i int) bool {
			return i%2 == 0
		})
	}
}

// BenchmarkLiveXYWhereSelect 测试 LiveXY 库的 WhereSelect 过滤性能
func BenchmarkLiveXYWhereSelect(b *testing.B) {
	var query = livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = livexy.WhereSelect(query, func(i int) (int, bool) {
			return i, i%2 == 0
		}).ToSlice()
	}
}

// BenchmarkAhmetbWhere 测试 go-linq (ahmetb) 库的过滤性能
func BenchmarkAhmetbWhere(b *testing.B) {
	var query = ahmetb.From(intData)
	for i := 0; i < b.N; i++ {
		var res []int
		query.Where(func(i interface{}) bool {
			return i.(int)%2 == 0
		}).ToSlice(&res)
	}
}

// BenchmarkLoWhere 测试 lo 库的过滤性能
func BenchmarkLoWhere(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Filter(intData, func(i int, _ int) bool {
			return i%2 == 0
		})
	}
}

// BenchmarkNativeWhere 测试原生 Go for 循环的过滤性能
func BenchmarkNativeWhere(b *testing.B) {
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

// BenchmarkLiveXYSelect 测试 LiveXY 库的映射性能
func BenchmarkLiveXYSelect(b *testing.B) {
	q := livexy.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Select(q, func(i int) int {
			return i * 2
		}).ToSlice()
	}
}

// BenchmarkLiveXYMapSlice 测试 LiveXY 库的切片级映射性能
func BenchmarkLiveXYMapSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = livexy.Map(intData, func(i int) int {
			return i * 2
		})
	}
}

// BenchmarkAhmetbSelect 测试 go-linq (ahmetb) 库的映射性能
func BenchmarkAhmetbSelect(b *testing.B) {
	query := ahmetb.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		query.Select(func(i interface{}) interface{} {
			return i.(int) * 2
		}).ToSlice(&res)
	}
}

// BenchmarkLoSelect 测试 lo 库的映射性能
func BenchmarkLoSelect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Map(intData, func(i int, _ int) int {
			return i * 2
		})
	}
}

// BenchmarkNativeSelect 测试原生 Go for 循环的映射性能
func BenchmarkNativeSelect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		for i, v := range intData {
			res[i] = v * 2
		}
	}
}

// --- 基准测试: 链式调用 (Where + Select) ---

// BenchmarkLiveXYChain 测试 LiveXY 库的链式调用 (过滤+映射) 性能
func BenchmarkLiveXYChain(b *testing.B) {
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

// BenchmarkLiveXYChainWhereSelect 测试 LiveXY 库的 WhereSelect 链式调用性能
func BenchmarkLiveXYChainWhereSelect(b *testing.B) {
	query := livexy.From(intData)
	for i := 0; i < b.N; i++ {
		_ = livexy.WhereSelect(query, func(i int) (int, bool) {
			return i * 2, i%2 == 0
		}).ToSlice()
	}
}

// BenchmarkLiveXYChainSlice 测试 LiveXY 库的切片级链式调用性能
func BenchmarkLiveXYChainSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := livexy.Where(intData, func(i int) bool {
			return i%2 == 0
		})
		_ = livexy.Map(filtered, func(i int) int {
			return i * 2
		})
	}
}

// BenchmarkAhmetbChain 测试 go-linq (ahmetb) 库的链式调用性能
func BenchmarkAhmetbChain(b *testing.B) {
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

// BenchmarkLoChain 测试 lo 库的链式调用性能
func BenchmarkLoChain(b *testing.B) {
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

// BenchmarkNativeChain 测试原生 Go 实现的链式处理性能
func BenchmarkNativeChain(b *testing.B) {
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

// BenchmarkLiveXYStruct 测试 LiveXY 库处理结构体切片的性能
func BenchmarkLiveXYStruct(b *testing.B) {
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

// BenchmarkLiveXYStructSlice 测试 LiveXY 库的切片级结构体处理性能
func BenchmarkLiveXYStructSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := livexy.Where(userList, func(u User) bool {
			return u.Age > 18
		})
		livexy.Map(filtered, func(u User) string {
			return u.Name
		})
	}
}

// BenchmarkAhmetbStruct 测试 go-linq (ahmetb) 库处理结构体切片的性能
func BenchmarkAhmetbStruct(b *testing.B) {
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

// BenchmarkLoStruct 测试 lo 库处理结构体切片的性能
func BenchmarkLoStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filtered := lo.Filter(userList, func(u User, _ int) bool {
			return u.Age > 18
		})
		lo.Map(filtered, func(u User, _ int) string {
			return u.Name
		})
	}
}

// BenchmarkNativeStruct 测试原生 Go 实现处理结构体的性能
func BenchmarkNativeStruct(b *testing.B) {
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

// BenchmarkLiveXYOneSort 测试 LiveXY 库的单级排序性能
func BenchmarkLiveXYOneSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		livexy.OrderBy(q, func(u User) int {
			return u.Age
		}).ToSlice()
	}
}

// BenchmarkNewOneSort 测试新实现 (Order API) 的单级排序性能
func BenchmarkNewOneSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(livexy.Asc(func(u User) int {
			return u.Age
		})).ToSlice()
	}
}

// BenchmarkAhmetbOneSort 测试 go-linq (ahmetb) 库的单级排序性能
func BenchmarkAhmetbOneSort(b *testing.B) {
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

// BenchmarkNativeOneSort 测试原生 Go sort.Slice 的单级排序性能
func BenchmarkNativeOneSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		sort.Slice(smallData, func(i, j int) bool { return smallData[i].Age < smallData[j].Age })
	}
}

// BenchmarkSlicesOneSort 测试原生 Go slices.SortFunc 的单级排序性能 (Go 1.21+)
func BenchmarkSlicesOneSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortFunc(smallData, func(a, b User) int {
			return a.Age - b.Age
		})
	}
}

// BenchmarkLiveXYTwoSort 测试 LiveXY 库的二级排序性能 (Age -> Gender)
func BenchmarkLiveXYTwoSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q2 := livexy.OrderBy(q, func(u User) int {
			return u.Age
		})
		livexy.ThenBy(q2, func(u User) int {
			return u.Gender
		}).ToSlice()
	}
}

// BenchmarkNewTwoSort 测试新实现 (Order API) 的二级排序性能
func BenchmarkNewTwoSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(livexy.Asc(func(u User) int {
			return u.Age
		})).Then(livexy.Asc(func(u User) int {
			return u.Gender
		})).ToSlice()
	}
}

// BenchmarkAhmetbTwoSort 测试 go-linq (ahmetb) 库的二级排序性能
func BenchmarkAhmetbTwoSort(b *testing.B) {
	smallData := userList[:1000]
	query := ahmetb.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []User
		query.OrderBy(func(i interface{}) interface{} {
			return i.(User).Age
		}).ThenBy(func(i interface{}) interface{} {
			return i.(User).Gender
		}).ToSlice(&res)
	}
}

// BenchmarkNativeTwoSort 测试原生 Go sort.Slice 的二级排序性能
func BenchmarkNativeTwoSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		sort.Slice(smallData, func(i, j int) bool {
			if smallData[i].Age != smallData[j].Age {
				return smallData[i].Age < smallData[j].Age
			}
			return smallData[i].Gender < smallData[j].Gender
		})
	}
}

// BenchmarkSlicesTwoSort 测试原生 Go slices.SortFunc 的二级排序性能
func BenchmarkSlicesTwoSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortFunc(smallData, func(a, b User) int {
			if a.Age != b.Age {
				return a.Age - b.Age
			}
			return a.Gender - b.Gender
		})
	}
}

// BenchmarkLiveXYThreeSort 测试 LiveXY 库的三级排序性能 (Age -> Gender -> ID)
func BenchmarkLiveXYThreeSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q2 := livexy.OrderBy(q, func(u User) int {
			return u.Age
		})
		q3 := livexy.ThenBy(q2, func(u User) int {
			return u.Gender
		})
		livexy.ThenBy(q3, func(u User) int {
			return u.ID
		}).ToSlice()
	}
}

// BenchmarkNewThreeSort 测试新实现 (Order API) 的三级排序性能
func BenchmarkNewThreeSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(livexy.Asc(func(u User) int {
			return u.Age
		})).Then(livexy.Asc(func(u User) int {
			return u.Gender
		})).Then(livexy.Asc(func(u User) int {
			return u.ID
		})).ToSlice()
	}
}

// BenchmarkNewThreeSort 测试新实现 (Order API) 的三级排序性能
func BenchmarkNew2ThreeSort(b *testing.B) {
	smallData := userList[:1000]
	q := livexy.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Order(func(a, b User) int {
			if c := cmp.Compare(a.Age, b.Age); c != 0 {
				return c
			}
			if c := cmp.Compare(a.Gender, b.Gender); c != 0 {
				return c
			}
			return cmp.Compare(a.ID, b.ID)
		}).ToSlice()
	}
}

// BenchmarkAhmetbThreeSort 测试 go-linq (ahmetb) 库的三级排序性能
func BenchmarkAhmetbThreeSort(b *testing.B) {
	smallData := userList[:1000]
	query := ahmetb.From(smallData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []User
		query.OrderBy(func(i interface{}) interface{} {
			return i.(User).Age
		}).ThenBy(func(i interface{}) interface{} {
			return i.(User).Gender
		}).ThenBy(func(i interface{}) interface{} {
			return i.(User).ID
		}).ToSlice(&res)
	}
}

// BenchmarkNativeThreeSort 测试原生 Go sort.Slice 的三级排序性能
func BenchmarkNativeThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		sort.Slice(smallData, func(i, j int) bool {
			if smallData[i].Age != smallData[j].Age {
				return smallData[i].Age < smallData[j].Age
			}
			if smallData[i].Gender != smallData[j].Gender {
				return smallData[i].Gender < smallData[j].Gender
			}
			return smallData[i].ID < smallData[j].ID
		})
	}
}

// BenchmarkNativeStableThreeSort 测试原生 Go sort.SliceStable 的三级排序性能
func BenchmarkNativeStableThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		sort.SliceStable(smallData, func(i, j int) bool {
			if smallData[i].Age != smallData[j].Age {
				return smallData[i].Age < smallData[j].Age
			}
			if smallData[i].Gender != smallData[j].Gender {
				return smallData[i].Gender < smallData[j].Gender
			}
			return smallData[i].ID < smallData[j].ID
		})
	}
}

// BenchmarkSlicesSortFuncThreeSort 测试原生 Go slices.SortFunc 的三级排序性能
func BenchmarkSlicesSortFuncThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortFunc(smallData, func(a, b User) int {
			if a.Age != b.Age {
				return a.Age - b.Age
			}
			if a.Gender != b.Gender {
				return a.Gender - b.Gender
			}
			return a.ID - b.ID
		})
	}
}

// BenchmarkSlicesSortCompareThreeSort 测试原生 Go slices.SortFunc 的三级排序性能
func BenchmarkSlicesSortCompareThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortFunc(smallData, func(a, b User) int {
			if c := cmp.Compare(a.Age, b.Age); c != 0 {
				return c
			}
			if c := cmp.Compare(a.Gender, b.Gender); c != 0 {
				return c
			}
			return cmp.Compare(a.ID, b.ID)
		})
	}
}

// BenchmarkSlicesStableFuncThreeSort 测试原生 Go slices.SortStableFunc 的三级排序性能
func BenchmarkSlicesStableFuncThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortStableFunc(smallData, func(a, b User) int {
			if a.Age != b.Age {
				return a.Age - b.Age
			}
			if a.Gender != b.Gender {
				return a.Gender - b.Gender
			}
			return a.ID - b.ID
		})
	}
}

// BenchmarkSlicesStableCompareThreeSort 测试原生 Go slices.SortStableFunc 的三级排序性能
func BenchmarkSlicesStableCompareThreeSort(b *testing.B) {
	smallData := make([]User, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(smallData, userList[:1000])
		slices.SortStableFunc(smallData, func(a, b User) int {
			if c := cmp.Compare(a.Age, b.Age); c != 0 {
				return c
			}
			if c := cmp.Compare(a.Gender, b.Gender); c != 0 {
				return c
			}
			return cmp.Compare(a.ID, b.ID)
		})
	}
}

// --- 基准测试: 去重 (Distinct) ---

// BenchmarkLiveXYDistinct 测试 LiveXY 库的去重性能
func BenchmarkLiveXYDistinct(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Distinct(query).ToSlice()
	}
}

// BenchmarkLiveXYDistinctSelect 测试 LiveXY 库的自定义键去重性能
func BenchmarkLiveXYDistinctSelect(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.DistinctSelect(query, func(i int) int {
			return i
		}).ToSlice()
	}
}

// BenchmarkLiveXYDistinctQuery 测试 LiveXY 库的 Query.Distinct 去重性能
func BenchmarkLiveXYDistinctQuery(b *testing.B) {
	query := livexy.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = query.Distinct().ToSlice()
	}
}

// BenchmarkLiveXYUniq 测试 LiveXY 库的切片级去重性能
func BenchmarkLiveXYUniq(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Uniq(duplicateData)
	}
}

// BenchmarkAhmetbDistinct 测试 go-linq (ahmetb) 库的去重性能
func BenchmarkAhmetbDistinct(b *testing.B) {
	query := ahmetb.From(duplicateData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		query.Distinct().ToSlice(&res)
	}
}

// BenchmarkLoDistinct 测试 lo 库的去重性能
func BenchmarkLoDistinct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Uniq(duplicateData)
	}
}

// BenchmarkNativeDistinct 测试使用 map 实现的原生去重性能
func BenchmarkNativeDistinct(b *testing.B) {
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

// BenchmarkLiveXYUnion 测试 LiveXY 库的并集去重性能
func BenchmarkLiveXYUnion(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Union(q1, q2).ToSlice()
	}
}

// BenchmarkLiveXYUnionSelect 测试 LiveXY 库的自定义键并集性能
func BenchmarkLiveXYUnionSelect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.UnionSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// BenchmarkLiveXYSliceUnion 测试 LiveXY 库的切片级并集性能
func BenchmarkLiveXYSliceUnion(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.SliceUnion(intData, intDataOther)
	}
}

// BenchmarkAhmetbUnion 测试 go-linq (ahmetb) 库的并集去重性能
func BenchmarkAhmetbUnion(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Union(q2).ToSlice(&res)
	}
}

// BenchmarkLoUnion 测试 lo 库的并集去重性能
func BenchmarkLoUnion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Union(intData, intDataOther)
	}
}

// BenchmarkNativeUnion 测试使用 map 实现的原生并集去重性能
func BenchmarkNativeUnion(b *testing.B) {
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

// BenchmarkLiveXYContains 测试 LiveXY 库的包含查询性能 (查找末尾元素)
func BenchmarkLiveXYContains(b *testing.B) {
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.SliceContains(intData, target)
	}
}

// BenchmarkAhmetbContains 测试 go-linq (ahmetb) 库的包含查询性能
func BenchmarkAhmetbContains(b *testing.B) {
	q := ahmetb.From(intData)
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Contains(target)
	}
}

// BenchmarkLoContains 测试 lo 库的包含查询性能
func BenchmarkLoContains(b *testing.B) {
	target := size - 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Contains(intData, target)
	}
}

// BenchmarkNativeContains 测试原生 Go for 循环的包含查询性能
func BenchmarkNativeContains(b *testing.B) {
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

// BenchmarkLiveXYEvery 测试 LiveXY 库的 Every 性能
func BenchmarkLiveXYEvery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Every(intData, intSubset)
	}
}

// BenchmarkAhmetbEvery 测试 go-linq (ahmetb) 库的 Every 性能 (组合实现)
func BenchmarkAhmetbEvery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ahmetb.From(intSubset).All(func(i interface{}) bool {
			return ahmetb.From(intData).Contains(i)
		})
	}
}

// BenchmarkLoEvery 测试 lo 库的 Every 性能
func BenchmarkLoEvery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Every(intData, intSubset)
	}
}

// BenchmarkNativeEvery 测试原生 Go 实现的 Every 性能
func BenchmarkNativeEvery(b *testing.B) {
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

// BenchmarkLiveXYSome 测试 LiveXY 库的 Some 性能
func BenchmarkLiveXYSome(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Some(intData, intSubset)
	}
}

// BenchmarkAhmetbSome 测试 go-linq (ahmetb) 库的 Some 性能 (组合实现)
func BenchmarkAhmetbSome(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ahmetb.From(intSubset).AnyWith(func(i interface{}) bool {
			return ahmetb.From(intData).Contains(i)
		})
	}
}

// BenchmarkLoSome 测试 lo 库的 Some 性能
func BenchmarkLoSome(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.Some(intData, intSubset)
	}
}

// BenchmarkNativeSome 测试原生 Go 实现的 Some 性能
func BenchmarkNativeSome(b *testing.B) {
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

// BenchmarkLiveXYOptimizedSome 测试 LiveXY 库的 Some 性能 (优化版)
func BenchmarkLiveXYOptimizedSome(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SomeOptimized(intData, intSubset)
	}
}

// --- 基准测试: 是否都不包含 (None) ---

// BenchmarkLiveXYNone 测试 LiveXY 库的 None 性能
// None 的逻辑是：集合 A 中没有任何元素属于集合 B。
func BenchmarkLiveXYNone(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.None(intData, intSubset)
	}
}

// BenchmarkAhmetbNone 测试 go-linq (ahmetb) 库的 None 性能 (组合实现)
func BenchmarkAhmetbNone(b *testing.B) {
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

// BenchmarkLoNone 测试 lo 库的 None 性能
func BenchmarkLoNone(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lo.None(intData, intSubset)
	}
}

// BenchmarkNativeNone 测试原生 Go 实现的 None 性能
func BenchmarkNativeNone(b *testing.B) {
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

// BenchmarkLiveXYConcat 测试 LiveXY 库的合并性能
func BenchmarkLiveXYConcat(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q1.Concat(q2).ToSlice()
	}
}

// BenchmarkLiveXYConcatSlice 测试 LiveXY 库的切片级合并性能
func BenchmarkLiveXYConcatSlice(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Concat(intData, intDataOther)
	}
}

// BenchmarkAhmetbConcat 测试 go-linq (ahmetb) 库的合并性能
func BenchmarkAhmetbConcat(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Concat(q2).ToSlice(&res)
	}
}

// BenchmarkLoConcat 测试 lo 库的合并性能
func BenchmarkLoConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Flatten([][]int{intData, intDataOther})
	}
}

// BenchmarkNativeConcat 测试原生 Go append 的合并性能
func BenchmarkNativeConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, 0, len(intData)+len(intDataOther))
		res = append(res, intData...)
		res = append(res, intDataOther...)
		_ = res
	}
}

// --- 基准测试: 交集 (Intersect) ---

// BenchmarkLiveXYIntersect 测试 LiveXY 库的交集性能
func BenchmarkLiveXYIntersect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Intersect(q1, q2).ToSlice()
	}
}

// BenchmarkLiveXYIntersectSelect 测试 LiveXY 库的自定义键交集性能
func BenchmarkLiveXYIntersectSelect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.IntersectSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// BenchmarkLiveXYSliceIntersect 测试 LiveXY 库的切片级交集性能
func BenchmarkLiveXYSliceIntersect(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.SliceIntersect(intData, intDataOther)
	}
}

// BenchmarkAhmetbIntersect 测试 go-linq (ahmetb) 库的交集性能
func BenchmarkAhmetbIntersect(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Intersect(q2).ToSlice(&res)
	}
}

// BenchmarkLoIntersect 测试 lo 库的交集性能
func BenchmarkLoIntersect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Intersect(intData, intDataOther)
	}
}

// BenchmarkNativeIntersect 测试原生 Go 使用 map 的交集性能
func BenchmarkNativeIntersect(b *testing.B) {
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

// BenchmarkLiveXYExcept 测试 LiveXY 库的差集性能
func BenchmarkLiveXYExcept(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Except(q1, q2).ToSlice()
	}
}

// BenchmarkLiveXYExceptSelect 测试 LiveXY 库的自定义键差集性能
func BenchmarkLiveXYExceptSelect(b *testing.B) {
	q1 := livexy.From(intData)
	q2 := livexy.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.ExceptSelect(q1, q2, func(i int) int {
			return i
		}).ToSlice()
	}
}

// BenchmarkLiveXYDifference 测试 LiveXY 库的切片级差集性能
func BenchmarkLiveXYDifference(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = livexy.Difference(intData, intDataOther)
	}
}

// BenchmarkAhmetbExcept 测试 go-linq (ahmetb) 库的差集性能
func BenchmarkAhmetbExcept(b *testing.B) {
	q1 := ahmetb.From(intData)
	q2 := ahmetb.From(intDataOther)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q1.Except(q2).ToSlice(&res)
	}
}

// BenchmarkLoExcept 测试 lo 库的差集性能 (Difference 只取左差集)
func BenchmarkLoExcept(b *testing.B) {
	for i := 0; i < b.N; i++ {
		left, _ := lo.Difference(intData, intDataOther)
		_ = left
	}
}

// BenchmarkNativeExcept 测试原生 Go 实现的差集性能
func BenchmarkNativeExcept(b *testing.B) {
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

// BenchmarkLiveXYReverse 测试 LiveXY 库的链式反转性能
func BenchmarkLiveXYReverse(b *testing.B) {
	q := livexy.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Reverse().ToSlice()
	}
}

// BenchmarkLiveXYReverseSlice 测试 LiveXY 库的切片级原地反转性能（仅操作局部副本）
func BenchmarkLiveXYReverseSlice(b *testing.B) {
	data := append([]int(nil), intData...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Reverse(data)
	}
}

// BenchmarkLiveXYCloneReverse 测试 LiveXY 库的克隆反转性能
func BenchmarkLiveXYCloneReverse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.CloneReverse(intData)
	}
}

// BenchmarkAhmetbReverse 测试 go-linq (ahmetb) 库的反转性能
func BenchmarkAhmetbReverse(b *testing.B) {
	q := ahmetb.From(intData)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res []int
		q.Reverse().ToSlice(&res)
	}
}

// BenchmarkLoReverse 测试 lo 库的反转性能
func BenchmarkLoReverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Reverse(intData)
	}
}

// BenchmarkMutableReverse 测试 lo 库的原地反转性能（仅操作局部副本）
func BenchmarkMutableReverse(b *testing.B) {
	data := append([]int(nil), intData...)
	for i := 0; i < b.N; i++ {
		mutable.Reverse(data)
	}
}

// BenchmarkNativeReverse 测试原生 Go 实现的反转性能
func BenchmarkNativeReverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		n := len(intData)
		for j := 0; j < n; j++ {
			res[j] = intData[n-1-j]
		}
		_ = res
	}
}

// BenchmarkNativeInplaceReverse 测试原生 Go 原地反转性能（仅操作局部副本）
func BenchmarkNativeInplaceReverse(b *testing.B) {
	data := append([]int(nil), intData...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := len(data)
		for j := 0; j < n/2; j++ {
			data[j], data[n-1-j] = data[n-1-j], data[j]
		}
	}
}

// BenchmarkLoCloneReverse 测试 lo 库带拷贝的反转性能 (为了公平对比)
func BenchmarkLoCloneReverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := make([]int, len(intData))
		copy(data, intData)
		_ = lo.Reverse(data)
	}
}

// --- 基准测试: 随机洗牌 (Shuffle) ---

// BenchmarkLiveXYShuffle 测试 LiveXY 库的随机洗牌性能 (含拷贝)
func BenchmarkLiveXYShuffle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = livexy.Shuffle(intData)
	}
}

// BenchmarkLoShuffle 测试 lo 库的随机洗牌性能（返回新切片）
func BenchmarkLoShuffle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = lo.Shuffle(intData)
	}
}

// BenchmarkMutableShuffle 测试 lo 库的随机洗牌性能（原地修改局部副本）
func BenchmarkMutableShuffle(b *testing.B) {
	data := append([]int(nil), intData...)
	for i := 0; i < b.N; i++ {
		mutable.Shuffle(data)
	}
}

// BenchmarkNativeShuffle 测试原生 Go 实现的随机洗牌性能 (含拷贝)
func BenchmarkNativeShuffle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := make([]int, len(intData))
		copy(res, intData)
		rand.Shuffle(len(res), func(i, j int) {
			res[i], res[j] = res[j], res[i]
		})
		_ = res
	}
}
