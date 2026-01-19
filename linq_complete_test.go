package linq

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// 数据源函数测试
// ============================================================================

// TestFromSlice 测试从切片创建 Query
func TestFromSlice(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).ToSlice()

	if len(result) != 5 {
		t.Errorf("Expected 5 items, got %d", len(result))
	}
	for i, v := range result {
		if v != nums[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, nums[i], v)
		}
	}
}

// TestFromChannel 测试从 Channel 创建 Query
func TestFromChannel(t *testing.T) {
	ch := make(chan int, 5)
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	result := FromChannel(ch).ToSlice()
	expected := []int{1, 2, 3, 4, 5}

	if len(result) != len(expected) {
		t.Errorf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestFromMap 测试从 Map 创建 Query
func TestFromMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	result := FromMap(m).ToSlice()

	if len(result) != 3 {
		t.Errorf("期望 3 个元素，实际得到 %d", len(result))
	}

	// 验证所有键值对都存在
	for _, kv := range result {
		if v, ok := m[kv.Key]; !ok || v != kv.Value {
			t.Errorf("意外的键值对: %v", kv)
		}
	}
}

// ============================================================================
// 过滤和分页测试
// ============================================================================

// TestWhere 测试条件过滤
func TestWhere(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := From(nums).Where(func(i int) bool { return i%2 == 0 }).ToSlice()

	expected := []int{2, 4, 6, 8, 10}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestSkip 测试跳过元素
func TestSkip(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Skip(2).ToSlice()

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestTake 测试获取前N个元素
func TestTake(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Take(3).ToSlice()

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// ============================================================================
// 集合操作测试
// ============================================================================

// TestAppendItem 测试追加单个元素
func TestAppendItem(t *testing.T) {
	nums := []int{1, 2, 3}
	result := From(nums).Append(4).ToSlice()

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestConcat 测试连接两个 Query
func TestConcat(t *testing.T) {
	nums1 := []int{1, 2, 3}
	nums2 := []int{4, 5, 6}
	result := From(nums1).Concat(From(nums2)).ToSlice()

	expected := []int{1, 2, 3, 4, 5, 6}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestPrepend 测试在前面添加元素
func TestPrepend(t *testing.T) {
	nums := []int{2, 3, 4}
	result := From(nums).Prepend(1).ToSlice()

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestDefaultIfEmpty 测试空集合默认值
func TestDefaultIfEmpty(t *testing.T) {
	empty := []int{}
	result := From(empty).DefaultIfEmpty(99).ToSlice()

	if len(result) != 1 || result[0] != 99 {
		t.Errorf("期望 [99]，实际得到 %v", result)
	}

	// 非空集合不应该添加默认值
	nums := []int{1, 2, 3}
	result2 := From(nums).DefaultIfEmpty(99).ToSlice()
	if len(result2) != 3 {
		t.Errorf("期望 3 个元素，实际得到 %d", len(result2))
	}
}

// TestDistinctMethod 测试 Query.Distinct 方法
func TestDistinctMethod(t *testing.T) {
	nums := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}
	result := From(nums).Distinct().ToSlice()

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestExcept 测试差集
func TestExcept(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := From(nums1).Except(From(nums2)).ToSlice()

	expected := []int{1, 2}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestIntersectMethod 测试 Query.Intersect 方法
func TestIntersectMethod(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := From(nums1).Intersect(From(nums2)).ToSlice()

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// ============================================================================
// 查询和断言测试
// ============================================================================

// TestIndexOfMethod 测试 Query.IndexOf 方法
func TestIndexOfMethod(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}
	idx := From(nums).IndexOf(func(i int) bool { return i == 30 })

	if idx != 2 {
		t.Errorf("期望索引 2，实际得到 %d", idx)
	}

	// 不存在的元素
	idx2 := From(nums).IndexOf(func(i int) bool { return i == 99 })
	if idx2 != -1 {
		t.Errorf("期望 -1，实际得到 %d", idx2)
	}
}

// TestAll 测试所有元素是否满足条件
func TestAll(t *testing.T) {
	nums := []int{2, 4, 6, 8, 10}
	result := From(nums).All(func(i int) bool { return i%2 == 0 })

	if !result {
		t.Error("所有偶数时期望 true")
	}

	nums2 := []int{2, 4, 5, 8, 10}
	result2 := From(nums2).All(func(i int) bool { return i%2 == 0 })
	if result2 {
		t.Error("当不全是偶数时期望 false")
	}
}

// TestAny 测试是否存在任何元素
func TestAny(t *testing.T) {
	nums := []int{1, 2, 3}
	if !From(nums).Any() {
		t.Error("非空切片期望 true")
	}

	empty := []int{}
	if From(empty).Any() {
		t.Error("空切片期望 false")
	}
}

// TestAnyWith 测试是否存在满足条件的元素
func TestAnyWith(t *testing.T) {
	nums := []int{1, 3, 5, 7, 8}
	result := From(nums).AnyWith(func(i int) bool { return i%2 == 0 })

	if !result {
		t.Error("期望 true，8 是偶数")
	}
}

// TestCountWith 测试满足条件的元素数量
func TestCountWith(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	count := From(nums).CountWith(func(i int) bool { return i%2 == 0 })

	if count != 5 {
		t.Errorf("期望 5 个偶数，实际得到 %d", count)
	}
}

// TestFirstWith 测试获取第一个满足条件的元素
func TestFirstWith(t *testing.T) {
	nums := []int{1, 3, 5, 6, 7, 8}
	result := From(nums).FirstWith(func(i int) bool { return i%2 == 0 })

	if result != 6 {
		t.Errorf("期望 6，实际得到 %d", result)
	}
}

// TestLast 测试获取最后一个元素
func TestLast(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Last()

	if result != 5 {
		t.Errorf("期望 5，实际得到 %d", result)
	}
}

// TestLastWith 测试获取最后一个满足条件的元素
func TestLastWith(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6}
	result := From(nums).LastWith(func(i int) bool { return i%2 == 0 })

	if result != 6 {
		t.Errorf("期望 6，实际得到 %d", result)
	}
}

// TestReverseMethod 测试 Query.Reverse 方法
func TestReverseMethod(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Reverse().ToSlice()

	expected := []int{5, 4, 3, 2, 1}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestSingle 测试获取单一元素
func TestSingle(t *testing.T) {
	single := []int{42}
	result := From(single).Single()

	if result != 42 {
		t.Errorf("期望 42，实际得到 %d", result)
	}

	// 多个元素应该返回零值
	multiple := []int{1, 2, 3}
	result2 := From(multiple).Single()
	if result2 != 0 {
		t.Errorf("期望 0 (多个元素时)，实际得到 %d", result2)
	}
}

// ============================================================================
// 遍历测试
// ============================================================================

// TestForEach 测试遍历
func TestForEach(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	sum := 0
	From(nums).ForEach(func(i int) bool {
		sum += i
		return true
	})

	if sum != 15 {
		t.Errorf("期望总和 15，实际得到 %d", sum)
	}
}

// TestForEachIndexed 测试带索引遍历
func TestForEachIndexed(t *testing.T) {
	nums := []int{10, 20, 30}
	var indices []int
	var values []int

	From(nums).ForEachIndexed(func(idx int, val int) bool {
		indices = append(indices, idx)
		values = append(values, val)
		return true
	})

	if len(indices) != 3 || indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Errorf("意外的索引: %v", indices)
	}
}

// ============================================================================
// 聚合函数测试
// ============================================================================

// TestSumAllTypes 测试各种类型的求和
func TestSumAllTypes(t *testing.T) {
	type Item struct{ Value int }
	items := []Item{{1}, {2}, {3}, {4}, {5}}

	sumInt := From(items).SumIntBy(func(i Item) int { return i.Value })
	if sumInt != 15 {
		t.Errorf("SumIntBy: 期望 15，实际得到 %d", sumInt)
	}

	sumInt64 := From(items).SumInt64By(func(i Item) int64 { return int64(i.Value) })
	if sumInt64 != 15 {
		t.Errorf("SumInt64By: 期望 15，实际得到 %d", sumInt64)
	}

	sumFloat := From(items).SumFloat64By(func(i Item) float64 { return float64(i.Value) })
	if sumFloat != 15.0 {
		t.Errorf("SumFloat64By: 期望 15.0，实际得到 %f", sumFloat)
	}
}

// TestAvgAllTypes 测试各种类型的平均值
func TestAvgAllTypes(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	avgInt := From(nums).AvgIntBy(func(i int) int { return i })
	if avgInt != 30.0 {
		t.Errorf("AvgIntBy: 期望 30.0，实际得到 %f", avgInt)
	}

	avgInt64 := From(nums).AvgInt64By(func(i int) int64 { return int64(i) })
	if avgInt64 != 30.0 {
		t.Errorf("AvgInt64By: 期望 30.0，实际得到 %f", avgInt64)
	}

	avgFloat := From(nums).AvgBy(func(i int) float64 { return float64(i) })
	if avgFloat != 30.0 {
		t.Errorf("AvgBy: 期望 30.0，实际得到 %f", avgFloat)
	}
}

// TestCount 测试计数
func TestCount(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	count := From(nums).Count()

	if count != 5 {
		t.Errorf("期望 5，实际得到 %d", count)
	}
}

// ============================================================================
// 输出函数测试
// ============================================================================

// TestToChannel 测试输出到 Channel
func TestToChannel(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	ch := make(chan int, 5)

	go From(nums).ToChannel(ch)

	var result []int
	for v := range ch {
		result = append(result, v)
	}

	if len(result) != 5 {
		t.Errorf("期望 5 个元素，实际得到 %d", len(result))
	}
}

// TestToMapSliceMethod 测试 Query.ToMapSlice 方法
func TestToMapSliceMethod(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{{"Alice", 30}, {"Bob", 25}}

	result := From(people).ToMapSlice(func(p Person) map[string]any {
		return map[string]any{"name": p.Name, "age": p.Age}
	})

	if len(result) != 2 {
		t.Errorf("期望 2 个元素，实际得到 %d", len(result))
	}
}

// ============================================================================
// 独立函数测试
// ============================================================================

// TestSelectFunction 测试 Select 函数
func TestSelectFunction(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := Select(From(nums), func(i int) string {
		return fmt.Sprintf("num_%d", i)
	}).ToSlice()

	expected := []string{"num_1", "num_2", "num_3", "num_4", "num_5"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %s，实际得到 %s", i, expected[i], v)
		}
	}
}

// TestDistinctFunction 测试 Distinct 函数
func TestDistinctFunction(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items := []Item{{1, "a"}, {2, "b"}, {1, "c"}, {3, "d"}, {2, "e"}}

	result := Distinct(From(items), func(i Item) int { return i.ID }).ToSlice()

	if len(result) != 3 {
		t.Errorf("期望 3 个不重复元素，实际得到 %d", len(result))
	}
}

// TestRange 测试 Range 函数
func TestRange(t *testing.T) {
	result := Range(1, 5).ToSlice()

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestRepeat 测试 Repeat 函数
func TestRepeat(t *testing.T) {
	result := Repeat("hello", 3).ToSlice()

	if len(result) != 3 {
		t.Fatalf("期望 3 个元素，实际得到 %d", len(result))
	}
	for _, v := range result {
		if v != "hello" {
			t.Errorf("期望 'hello'，实际得到 '%s'", v)
		}
	}
}

// TestToMapFunction 测试 ToMap 函数
func TestToMapFunction(t *testing.T) {
	type Item struct {
		Key   string
		Value int
	}
	items := []Item{{"a", 1}, {"b", 2}, {"c", 3}}

	result := ToMap(From(items), func(i Item) string { return i.Key })

	if result["a"].Value != 1 || result["b"].Value != 2 || result["c"].Value != 3 {
		t.Errorf("意外的 map: %v", result)
	}
}

// ============================================================================
// 切片工具函数测试
// ============================================================================

// TestUniq 测试去重
func TestUniq(t *testing.T) {
	nums := []int{1, 2, 2, 3, 3, 3, 4}
	result := Uniq(nums)

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestContains 测试包含
func TestContains(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}

	if !Contains(nums, 3) {
		t.Error("3 期望返回 true")
	}
	if Contains(nums, 99) {
		t.Error("99 期望返回 false")
	}
}

// TestIndexOfFunction 测试 IndexOf 函数
func TestIndexOfFunction(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	if IndexOf(nums, 30) != 2 {
		t.Error("30 期望索引 2")
	}
	if IndexOf(nums, 99) != -1 {
		t.Error("99 期望 -1")
	}
}

// TestLastIndexOf 测试 LastIndexOf 函数
func TestLastIndexOf(t *testing.T) {
	nums := []int{1, 2, 3, 2, 1}

	if LastIndexOf(nums, 2) != 3 {
		t.Error("2 期望最后索引 3")
	}
}

// TestShuffle 测试随机打乱
func TestShuffle(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	original := make([]int, len(nums))
	copy(original, nums)

	result := Shuffle(nums)

	// 验证长度相同
	if len(result) != len(nums) {
		t.Errorf("期望相同长度，实际得到 %d", len(result))
	}

	// 验证原数组未被修改
	for i, v := range nums {
		if v != original[i] {
			t.Error("原数组被修改")
			break
		}
	}
}

// TestReverseFunction 测试 Reverse 函数
func TestReverseFunction(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := Reverse(nums)

	expected := []int{5, 4, 3, 2, 1}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestMinMax 测试最小最大值
func TestMinMax(t *testing.T) {
	if Min(3, 1, 4, 1, 5) != 1 {
		t.Error("Min 失败")
	}
	if Max(3, 1, 4, 1, 5) != 5 {
		t.Error("Max 失败")
	}

	// 空切片返回零值
	if Min[int]() != 0 {
		t.Error("空 Min 应该返回 0")
	}
}

// TestSumFunction 测试 Sum 函数
func TestSumFunction(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := Sum(nums)

	if result != 15 {
		t.Errorf("期望 15，实际得到 %d", result)
	}

	floats := []float64{1.5, 2.5, 3.0}
	resultFloat := Sum(floats)
	if resultFloat != 7.0 {
		t.Errorf("期望 7.0，实际得到 %f", resultFloat)
	}
}

// TestEverySomeNone 测试集合判断
func TestEverySomeNone(t *testing.T) {
	list := []int{1, 2, 3, 4, 5}

	// Every: 所有元素都在 list 中
	if !Every(list, []int{1, 3, 5}) {
		t.Error("Every 失败")
	}
	if Every(list, []int{1, 6}) {
		t.Error("Every 对于 6 应该返回 false")
	}

	// Some: 至少一个元素在 list 中
	if !Some(list, []int{5, 6, 7}) {
		t.Error("Some 失败")
	}
	if Some(list, []int{6, 7, 8}) {
		t.Error("Some 应该返回 false")
	}

	// None: 没有元素在 list 中
	if !None(list, []int{6, 7, 8}) {
		t.Error("None 失败")
	}
	if None(list, []int{5, 6, 7}) {
		t.Error("None 对于 5 应该返回 false")
	}
}

// TestIntersectFunction 测试 Intersect 函数
func TestIntersectFunction(t *testing.T) {
	list1 := []int{1, 2, 3, 4, 5}
	list2 := []int{3, 4, 5, 6, 7}
	result := Intersect(list1, list2)

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestDifference 测试差异
func TestDifference(t *testing.T) {
	list1 := []int{1, 2, 3, 4, 5}
	list2 := []int{3, 4, 5, 6, 7}
	left, right := Difference(list1, list2)

	if len(left) != 2 || left[0] != 1 || left[1] != 2 {
		t.Errorf("左差集失败: %v", left)
	}
	if len(right) != 2 || right[0] != 6 || right[1] != 7 {
		t.Errorf("右差集失败: %v", right)
	}
}

// TestUnionFunction 测试 Union 函数
func TestUnionFunction(t *testing.T) {
	list1 := []int{1, 2, 3}
	list2 := []int{3, 4, 5}
	result := Union(list1, list2)

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestNoEmpty 测试过滤空值
func TestNoEmpty(t *testing.T) {
	strs := []string{"a", "", "b", "", "c"}
	result := NoEmpty(strs)

	expected := []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestGtZero 测试过滤大于零的值
func TestGtZero(t *testing.T) {
	nums := []int{-2, -1, 0, 1, 2, 3}
	result := GtZero(nums)

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestRand 测试随机取元素
func TestRand(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := Rand(nums, 3)

	if len(result) != 3 {
		t.Errorf("期望 3 个元素，实际得到 %d", len(result))
	}

	// 验证结果都在原数组中
	for _, v := range result {
		if !Contains(nums, v) {
			t.Errorf("意外的值: %d", v)
		}
	}
}

// ============================================================================
// 工具函数测试
// ============================================================================

// TestDefault 测试默认值
func TestDefault(t *testing.T) {
	if Default(0, 42) != 42 {
		t.Error("Default 对于 0 应该返回 42")
	}
	if Default(10, 42) != 10 {
		t.Error("Default 对于非零值应该返回 10")
	}
	if Default("", "default") != "default" {
		t.Error("Default 对于空字符串应该返回 'default'")
	}
}

// TestEmpty 测试获取零值
func TestEmpty(t *testing.T) {
	if Empty[int]() != 0 {
		t.Error("Empty[int] 应该是 0")
	}
	if Empty[string]() != "" {
		t.Error("Empty[string] 应该是 ''")
	}
}

// TestIsEmptyIsNotEmpty 测试空值判断
func TestIsEmptyIsNotEmpty(t *testing.T) {
	if !IsEmpty(0) {
		t.Error("IsEmpty(0) 应该是 true")
	}
	if !IsEmpty("") {
		t.Error("IsEmpty('') 应该是 true")
	}
	if IsEmpty(1) {
		t.Error("IsEmpty(1) 应该是 false")
	}

	if IsNotEmpty(0) {
		t.Error("IsNotEmpty(0) 应该是 false")
	}
	if !IsNotEmpty(1) {
		t.Error("IsNotEmpty(1) 应该是 true")
	}
}

// TestTry 测试 Try 函数
func TestTry(t *testing.T) {
	// 成功的情况
	success := Try(func() error { return nil })
	if !success {
		t.Error("Try 成功时应该返回 true")
	}

	// 失败的情况
	failure := Try(func() error { return fmt.Errorf("error") })
	if failure {
		t.Error("Try 错误时应该返回 false")
	}

	// Panic 的情况
	panicCase := Try(func() error { panic("panic") })
	if panicCase {
		t.Error("Try panic 时应该返回 false")
	}
}

// TestTryCatch 测试 TryCatch 函数
func TestTryCatch(t *testing.T) {
	caught := false
	TryCatch(func() error {
		panic("test panic")
	}, func() {
		caught = true
	})

	if !caught {
		t.Error("应该调用 Catch 函数")
	}
}

// TestIF 测试三目运算
func TestIF(t *testing.T) {
	if IF(true, "yes", "no") != "yes" {
		t.Error("IF(true) 应该返回 'yes'")
	}
	if IF(false, "yes", "no") != "no" {
		t.Error("IF(false) 应该返回 'no'")
	}
	if IF(1 > 0, 100, 200) != 100 {
		t.Error("IF(1>0) 应该返回 100")
	}
}

// ============================================================================
// 额外覆盖测试
// ============================================================================

// TestExceptComparable 测试优化的差集
func TestExceptComparable(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := ExceptComparable(From(nums1), From(nums2)).ToSlice()

	expected := []int{1, 2}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestIntersectComparable 测试优化的交集
func TestIntersectComparable(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := IntersectComparable(From(nums1), From(nums2)).ToSlice()

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestExceptBy 测试 ExceptBy 函数
func TestExceptBy(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items1 := []Item{{1, "a"}, {2, "b"}, {3, "c"}}
	items2 := []Item{{2, "x"}, {3, "y"}}

	// ExceptBy 返回的是 selector 的结果类型 (int)，不是原始 Item
	result := ExceptBy(From(items1), From(items2), func(i Item) int { return i.ID }).ToSlice()

	// 只有 ID=1 不在 items2 中
	if len(result) != 1 || result[0] != 1 {
		t.Errorf("期望 [1]，实际得到 %v", result)
	}
}

// TestIntersectBy 测试 IntersectBy 函数
func TestIntersectBy(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items1 := []Item{{1, "a"}, {2, "b"}, {3, "c"}}
	items2 := []Item{{2, "x"}, {3, "y"}}

	// IntersectBy 返回的是 selector 的结果类型 (int)
	result := IntersectBy(From(items1), From(items2), func(i Item) int { return i.ID }).ToSlice()

	// ID=2 和 ID=3 在两个集合中都存在
	if len(result) != 2 {
		t.Errorf("期望 2 个元素，实际得到 %d: %v", len(result), result)
	}
}

// TestSkipMoreThanLength 测试 Skip 超过长度
func TestSkipMoreThanLength(t *testing.T) {
	nums := []int{1, 2, 3}
	result := From(nums).Skip(10).ToSlice()

	if len(result) != 0 {
		t.Errorf("期望空切片，实际得到 %d 个元素", len(result))
	}
}

// TestTakeMoreThanLength 测试 Take 超过长度
func TestTakeMoreThanLength(t *testing.T) {
	nums := []int{1, 2, 3}
	result := From(nums).Take(10).ToSlice()

	if len(result) != 3 {
		t.Errorf("期望 3 个元素，实际得到 %d", len(result))
	}
}

// TestAnyWithNoMatch 测试 AnyWith 无匹配
func TestAnyWithNoMatch(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9}
	result := From(nums).AnyWith(func(i int) bool { return i%2 == 0 })

	if result {
		t.Error("期望 false，没有偶数")
	}
}

// TestFirstWithNoMatch 测试 FirstWith 无匹配
func TestFirstWithNoMatch(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9}
	result := From(nums).FirstWith(func(i int) bool { return i > 100 })

	if result != 0 {
		t.Errorf("无匹配时期望 0，实际得到 %d", result)
	}
}

// TestForEachEarlyExit 测试 ForEach 提前退出
func TestForEachEarlyExit(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	count := 0
	From(nums).ForEach(func(i int) bool {
		count++
		return i < 3 // 在 3 时停止
	})

	if count != 3 {
		t.Errorf("期望 3 次迭代，实际得到 %d", count)
	}
}

// TestForEachIndexedEarlyExit 测试 ForEachIndexed 提前退出
func TestForEachIndexedEarlyExit(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}
	lastIdx := -1
	From(nums).ForEachIndexed(func(idx int, val int) bool {
		lastIdx = idx
		return idx < 2
	})

	if lastIdx != 2 {
		t.Errorf("期望最后索引 2，实际得到 %d", lastIdx)
	}
}

// TestUnionWithDuplicates 测试 Union 去重
func TestUnionWithDuplicates(t *testing.T) {
	nums1 := []int{1, 2, 2, 3}
	nums2 := []int{3, 4, 4, 5}
	result := From(nums1).Union(From(nums2)).ToSlice()

	// 应该去重
	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestTryWithRetry 测试 Try 重试
func TestTryWithRetry(t *testing.T) {
	attempts := 0
	success := Try(func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("error")
		}
		return nil
	}, 5) // 最多重试 5 次

	if !success {
		t.Error("期望重试后成功")
	}
	if attempts != 3 {
		t.Errorf("期望 3 次尝试，实际得到 %d", attempts)
	}
}

// TestSumInt8By 测试 int8 求和
func TestSumInt8To64(t *testing.T) {
	type Num struct{ Val int }
	nums := []Num{{1}, {2}, {3}}

	sum8 := From(nums).SumInt8By(func(n Num) int8 { return int8(n.Val) })
	if sum8 != 6 {
		t.Errorf("SumInt8By: 期望 6，实际得到 %d", sum8)
	}

	sum16 := From(nums).SumInt16By(func(n Num) int16 { return int16(n.Val) })
	if sum16 != 6 {
		t.Errorf("SumInt16By: 期望 6，实际得到 %d", sum16)
	}

	sum32 := From(nums).SumInt32By(func(n Num) int32 { return int32(n.Val) })
	if sum32 != 6 {
		t.Errorf("SumInt32By: 期望 6，实际得到 %d", sum32)
	}
}

// TestSumUIntTypes 测试无符号整数求和
func TestSumUIntTypes(t *testing.T) {
	type Num struct{ Val uint }
	nums := []Num{{1}, {2}, {3}}

	sumU := From(nums).SumUIntBy(func(n Num) uint { return n.Val })
	if sumU != 6 {
		t.Errorf("SumUIntBy: 期望 6，实际得到 %d", sumU)
	}

	sumU8 := From(nums).SumUInt8By(func(n Num) uint8 { return uint8(n.Val) })
	if sumU8 != 6 {
		t.Errorf("SumUInt8By: 期望 6，实际得到 %d", sumU8)
	}

	sumU16 := From(nums).SumUInt16By(func(n Num) uint16 { return uint16(n.Val) })
	if sumU16 != 6 {
		t.Errorf("SumUInt16By: 期望 6，实际得到 %d", sumU16)
	}

	sumU32 := From(nums).SumUInt32By(func(n Num) uint32 { return uint32(n.Val) })
	if sumU32 != 6 {
		t.Errorf("SumUInt32By: 期望 6，实际得到 %d", sumU32)
	}

	sumU64 := From(nums).SumUInt64By(func(n Num) uint64 { return uint64(n.Val) })
	if sumU64 != 6 {
		t.Errorf("SumUInt64By: 期望 6，实际得到 %d", sumU64)
	}
}

// TestSumFloat32 测试 float32 求和
func TestSumFloat32(t *testing.T) {
	type Num struct{ Val float32 }
	nums := []Num{{1.5}, {2.5}, {3.0}}

	sum := From(nums).SumFloat32By(func(n Num) float32 { return n.Val })
	if sum != 7.0 {
		t.Errorf("期望 7.0，实际得到 %f", sum)
	}
}

// TestSelectAsyncCtx 测试带 Context 的并发 Select
func TestSelectAsyncCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 100
	nums := Range(0, count)

	// 我们只想要前 10 个
	result := SelectAsyncCtx(ctx, nums, 5, func(i int) int {
		return i * 2
	}).Take(10).ToSlice()

	// 显式取消，模拟提前退出
	cancel()

	if len(result) != 10 {
		t.Errorf("期望 10 个元素，实际得到 %d", len(result))
	}

	// 验证结果内容
	for i, v := range result {
		if v%2 != 0 {
			t.Errorf("索引 %d: 期望偶数，实际得到 %d", i, v)
		}
		if v < 0 || v >= count*2 {
			t.Errorf("索引 %d: 值 %d 超出范围", i, v)
		}
	}

	// 等待让 goroutine 退出
	time.Sleep(50 * time.Millisecond)
}

// TestTakeWhile 测试 TakeWhile
func TestTakeWhile(t *testing.T) {
	nums := []int{1, 2, 3, 4, 1, 2}
	result := From(nums).TakeWhile(func(i int) bool { return i < 4 }).ToSlice()

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestSkipWhile 测试 SkipWhile
func TestSkipWhile(t *testing.T) {
	nums := []int{1, 2, 3, 4, 1, 2}
	result := From(nums).SkipWhile(func(i int) bool { return i < 3 }).ToSlice()

	expected := []int{3, 4, 1, 2}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}

// TestForEachParallelCtx 测试 ForEachParallelCtx
func TestForEachParallelCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nums := Range(0, 100)
	var processed atomic.Int32

	// 启动一个耗时的处理并在中途取消
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	nums.ForEachParallelCtx(ctx, 10, func(i int) {
		time.Sleep(100 * time.Millisecond)
		processed.Add(1)
	})

	// 应该只有一部分被处理（可能少于100），且不应该永久阻塞
	if processed.Load() == 100 {
		t.Error("期望少于 100 个元素被处理")
	}
}
