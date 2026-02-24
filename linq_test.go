// go test -v linq_test.go linq.go

package linq

import (
	"fmt"
	"sync"
	"testing"
	"time"
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

// TestLinqWhere 测试LINQ条件
func TestLinqWhere(t *testing.T) {
	var query = From(members).
		Where(func(m *BMember) bool { return m.Age == 28 })
	fmt.Printf("年龄28的人数: %+v \n", query.Count())
	query = query.Where(func(m *BMember) bool { return m.Sex == 1 })
	fmt.Printf("年龄28的男生人数: %+v \n", query.Count())
	fmt.Printf("年龄28的男生姓名: %+v \n", query.First().Name)
	fmt.Printf("年龄28的男生姓名: %+v \n", query.Where(func(m *BMember) bool { return m.Sex == 2 }).DefaultIfEmpty(&BMember{}).First().Name)
}

// TestSum 测试数值聚合函数 (Sum, Avg, Min, Max)
func TestSum(t *testing.T) {
	fmt.Printf("年龄总和: %+v \n", SumBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("平均年龄: %+v \n", AverageBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("年龄总和: %+v \n", SumBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("最小年龄: %+v \n", MinBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("最大年龄: %+v \n", MaxBy(From(members), func(m *BMember) int { return m.Age }))
}

// TestPage 测试分页功能 (Page, Skip, Take)
func TestPage(t *testing.T) {
	page, pageSize := 1, 3
	out1 := From(members).Skip((page - 1) * pageSize).Take(pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
	page = 2
	out1 = From(members).Page(page, pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
}

// TestUnion 测试集合并集 (Union)
func TestSliceUnion(t *testing.T) {
	out := Union(From(members), From(members)).ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
}

// TestOrder 测试排序功能 (OrderBy, ThenBy)
func TestOrder(t *testing.T) {
	query := From(members)
	query = OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	query = ThenBy(query, func(m *BMember) int { return m.Age })
	out4 := query.ToSlice()
	for _, v := range out4 {
		fmt.Printf("%+v \n", v)
	}
}

// TestOrder2 测试排序功能 (OrderBy, ThenBy)
func TestOrder2(t *testing.T) {
	out := From(members).
		Order(Desc(func(m *BMember) int8 { return m.Sex })).
		Then(Asc(func(m *BMember) int { return m.Age })).
		ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
}

// TestFrom 测试基本查询操作和组合 (From, Where, Select, GroupBy)
func TestFrom(t *testing.T) {
	out := From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 }).
		ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
	out2 := Select(
		From(out),
		func(m *BMember) *SMember { return &SMember{ID: m.ID, Name: m.Name} },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
	out3 := GroupBy(
		From(members),
		func(m *BMember) int8 { return m.Sex },
	).ToSlice()
	for _, v := range out3 {
		fmt.Printf("Key: %v, Value: %v \n", v.Key, v.Value)
	}
	out4 := GroupBySelect(
		From(members),
		func(m *BMember) int8 { return m.Sex },
		func(m *BMember) *BMember { return m },
	).ToSlice()
	for _, v := range out4 {
		fmt.Printf("Key: %v, Value: %v \n", v.Key, v.Value)
	}
}

// TestWhereSelect 测试过滤和类型转换
func TestWhereSelect(t *testing.T) {
	out2 := WhereSelect(
		From(members),
		func(m *BMember) (*SMember, bool) { return nil, true },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
}

// TestHasOrder 测试排序状态检查 (HasOrder)
func TestHasOrder(t *testing.T) {
	query := From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 })
	fmt.Printf("%+v \n", query.HasOrder())
	query = OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	fmt.Printf("%+v \n", query.HasOrder())
}

// TestFirst 测试获取第一个元素 (First, DefaultIfEmpty)
func TestFirst(t *testing.T) {
	fmt.Println(1, From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).DefaultIfEmpty(&BMember{}).First())
	fmt.Println(2, From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).First())
}

// TestFromString 测试字符串源 (FromString)
func TestFromString(t *testing.T) {
	str := "Hello, 世界! 🌍"
	q := FromString(str)
	slice := q.ToSlice()
	expected := []string{"H", "e", "l", "l", "o", ",", " ", "世", "界", "!", " ", "🌍"}
	if len(slice) != len(expected) {
		t.Fatalf("期望长度 %d，实际得到 %d", len(expected), len(slice))
	}
	for i, v := range slice {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %s，实际得到 %s", i, expected[i], v)
		}
	}
}

// TestMinMaxBy 测试自定义比较的最值查找 (MinBy, MaxBy)
func TestMinMaxBy(t *testing.T) {
	// 负数 MinBy 的测试用例
	nums := []int{-5, -2, -9, -1}
	min := MinBy(From(nums), func(i int) int { return i })
	if min != -9 {
		t.Errorf("期望最小值 -9，实际得到 %d", min)
	}

	max := MaxBy(From(nums), func(i int) int { return i })
	if max != -1 {
		t.Errorf("期望最大值 -1，实际得到 %d", max)
	}

	// 混合 0 的 MinBy 测试用例
	nums2 := []int{5, 0, 2}
	min2 := MinBy(From(nums2), func(i int) int { return i })
	if min2 != 0 {
	}
}

// TestAppendTo 测试将结果追加到切片 (AppendTo)
func TestAppendTo(t *testing.T) {
	nums := []int{1, 2, 3}
	buffer := make([]int, 0, 10)
	// 添加初始垃圾数据以确保追加正确
	buffer = append(buffer, 99)

	result := From(nums).AppendTo(buffer)

	expected := []int{99, 1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("期望长度 %d，实际得到 %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
	// 验证是否是相同的底层数组（如果容量允许）
	if &result[0] != &buffer[0] {
		t.Log("警告: 切片重新分配了，如果容量改变这是预期的，但请检查逻辑")
	}
}

// TestForEachParallel 测试并发遍历 (ForEachParallel)
func TestForEachParallel(t *testing.T) {
	count := 100
	nums := Range(0, count).ToSlice()
	var mu sync.Mutex
	processed := make(map[int]struct{})

	From(nums).ForEachParallel(10, func(i int) {
		mu.Lock()
		processed[i] = struct{}{}
		mu.Unlock()
		time.Sleep(1 * time.Millisecond) // 模拟工作
	})

	if len(processed) != count {
		t.Errorf("期望 %d 个处理项，实际得到 %d", count, len(processed))
	}
}

// TestSelectAsync 测试异步选择 (SelectAsync)
func TestSelectAsync(t *testing.T) {
	count := 50
	nums := Range(0, count)

	// SelectAsync 顺序不保证，所以我们检查存在性
	result := SelectAsync(nums, 5, func(i int) int {
		time.Sleep(1 * time.Millisecond)
		return i * 2
	}).ToSlice()

	if len(result) != count {
		t.Fatalf("期望 %d 个元素，实际得到 %d", count, len(result))
	}

	perm := make(map[int]bool)
	for _, v := range result {
		perm[v] = true
	}

	for i := 0; i < count; i++ {
		if !perm[i*2] {
			t.Errorf("缺失期望值 %d", i*2)
		}
	}
}

// TestPredicates 测试断言函数 (Any, All, Count, CountWith)
func TestPredicates(t *testing.T) {
	q := From(members)

	if !q.Any() {
		t.Error("Any() 应该返回 true")
	}
	if !q.All(func(m *BMember) bool { return m.Age > 0 }) {
		t.Error("All(Age > 0) 应该返回 true")
	}
	if q.Count() != 4 {
		t.Errorf("Count() 应该为 4，实际为 %d", q.Count())
	}
	count29 := q.CountWith(func(m *BMember) bool { return m.Age == 29 })
	if count29 != 2 {
		t.Errorf("CountWith(Age=29) 应该为 2，实际为 %d", count29)
	}
}

// TestElementAccess 测试元素访问 (Last, Single)
func TestElementAccess(t *testing.T) {
	q := From(members)

	last := q.Last()
	if last.Name != "老六" {
		t.Errorf("Last() 应该是 老六，实际为 %s", last.Name)
	}

	// 测试 Single (需要构造只有一个元素的 Query)
	singleQ := From(members).Where(func(m *BMember) bool { return m.Name == "张三" })
	single := singleQ.Single()
	if single == nil || single.Name != "张三" {
		t.Error("Single() 应该返回 张三")
	}
}

// TestToMapUsage 测试映射转换 (ToMapSlice, ToMap)
func TestToMapUsage(t *testing.T) {
	// 测试 Q.ToMapSlice
	maps := From(members).ToMapSlice(func(m *BMember) map[string]*BMember {
		return map[string]*BMember{m.Name: m}
	})
	if len(maps) != 4 {
		t.Errorf("ToMapSlice 应该返回 4 个元素")
	}
	if maps[0]["张三"].Name != "张三" {
		t.Errorf("第一个元素的 Name 应该是 张三")
	}

	// 测试 linq.ToMap
	dict := ToMap(From(members), func(m *BMember) int64 {
		return m.ID
	})
	if len(dict) != 4 {
		t.Errorf("ToMap 应该返回 4 个元素")
	}
	if dict[1].Name != "张三" {
		t.Errorf("ID为1的元素应该是 张三")
	}
}

// TestWhileOperations 测试 TakeWhile 和 SkipWhile
func TestWhileOperations(t *testing.T) {
	// members: 28, 28, 29, 29
	// TakeWhile Age < 29 => 应该是前两个
	take := From(members).TakeWhile(func(m *BMember) bool {
		return m.Age < 29
	}).ToSlice()

	if len(take) != 2 {
		t.Errorf("TakeWhile 应该返回 2 个元素，实际 %d", len(take))
	}
	if take[0].Name != "张三" || take[1].Name != "李四" {
		t.Error("TakeWhile 结果不匹配")
	}

	// SkipWhile Age < 29 => 应该是后两个
	skip := From(members).SkipWhile(func(m *BMember) bool {
		return m.Age < 29
	}).ToSlice()

	if len(skip) != 2 {
		t.Errorf("SkipWhile 应该返回 2 个元素，实际 %d", len(skip))
	}
	if skip[0].Name != "王五" || skip[1].Name != "老六" {
		t.Error("SkipWhile 结果不匹配")
	}
}

// TestSetOperations 测试集合操作 (Concat, Prepend, Append)
func TestSetOperations(t *testing.T) {
	q := From(members) // 4 items

	// Append
	q2 := q.Append(&BMember{ID: 5, Name: "小七"})
	if q2.Count() != 5 {
		t.Errorf("Append 后数量应该是 5")
	}
	if q2.Last().Name != "小七" {
		t.Errorf("最后一个应该是 小七")
	}

	// Prepend
	q3 := q.Prepend(&BMember{ID: 0, Name: "老祖"})
	if q3.Count() != 5 {
		t.Errorf("Prepend 后数量应该是 5")
	}
	if q3.First().Name != "老祖" {
		t.Errorf("第一个应该是 老祖")
	}

	// Concat
	q4 := q.Concat(From(members))
	if q4.Count() != 8 {
		t.Errorf("Concat 后数量应该是 8")
	}
}
