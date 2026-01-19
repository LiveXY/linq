package linq

import (
	"fmt"
	"testing"
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
			t.Errorf("Index %d: expected %d, got %d", i, nums[i], v)
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
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestFromMap 测试从 Map 创建 Query
func TestFromMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	result := FromMap(m).ToSlice()

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// 验证所有键值对都存在
	for _, kv := range result {
		if v, ok := m[kv.Key]; !ok || v != kv.Value {
			t.Errorf("Unexpected key-value: %v", kv)
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
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Index %d: expected %d, got %d", i, expected[i], v)
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
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestConcat 测试连接两个 Query
func TestConcat(t *testing.T) {
	nums1 := []int{1, 2, 3}
	nums2 := []int{4, 5, 6}
	result := From(nums1).Concat(From(nums2)).ToSlice()

	expected := []int{1, 2, 3, 4, 5, 6}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
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
		t.Errorf("Expected [99], got %v", result)
	}

	// 非空集合不应该添加默认值
	nums := []int{1, 2, 3}
	result2 := From(nums).DefaultIfEmpty(99).ToSlice()
	if len(result2) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result2))
	}
}

// TestDistinctMethod 测试 Query.Distinct 方法
func TestDistinctMethod(t *testing.T) {
	nums := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}
	result := From(nums).Distinct().ToSlice()

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestExcept 测试差集
func TestExcept(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := From(nums1).Except(From(nums2)).ToSlice()

	expected := []int{1, 2}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestIntersectMethod 测试 Query.Intersect 方法
func TestIntersectMethod(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := From(nums1).Intersect(From(nums2)).ToSlice()

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
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
		t.Errorf("Expected index 2, got %d", idx)
	}

	// 不存在的元素
	idx2 := From(nums).IndexOf(func(i int) bool { return i == 99 })
	if idx2 != -1 {
		t.Errorf("Expected -1, got %d", idx2)
	}
}

// TestAll 测试所有元素是否满足条件
func TestAll(t *testing.T) {
	nums := []int{2, 4, 6, 8, 10}
	result := From(nums).All(func(i int) bool { return i%2 == 0 })

	if !result {
		t.Error("Expected true for all even numbers")
	}

	nums2 := []int{2, 4, 5, 8, 10}
	result2 := From(nums2).All(func(i int) bool { return i%2 == 0 })
	if result2 {
		t.Error("Expected false when not all are even")
	}
}

// TestAny 测试是否存在任何元素
func TestAny(t *testing.T) {
	nums := []int{1, 2, 3}
	if !From(nums).Any() {
		t.Error("Expected true for non-empty slice")
	}

	empty := []int{}
	if From(empty).Any() {
		t.Error("Expected false for empty slice")
	}
}

// TestAnyWith 测试是否存在满足条件的元素
func TestAnyWith(t *testing.T) {
	nums := []int{1, 3, 5, 7, 8}
	result := From(nums).AnyWith(func(i int) bool { return i%2 == 0 })

	if !result {
		t.Error("Expected true, 8 is even")
	}
}

// TestCountWith 测试满足条件的元素数量
func TestCountWith(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	count := From(nums).CountWith(func(i int) bool { return i%2 == 0 })

	if count != 5 {
		t.Errorf("Expected 5 even numbers, got %d", count)
	}
}

// TestFirstWith 测试获取第一个满足条件的元素
func TestFirstWith(t *testing.T) {
	nums := []int{1, 3, 5, 6, 7, 8}
	result := From(nums).FirstWith(func(i int) bool { return i%2 == 0 })

	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
}

// TestLast 测试获取最后一个元素
func TestLast(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Last()

	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}
}

// TestLastWith 测试获取最后一个满足条件的元素
func TestLastWith(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6}
	result := From(nums).LastWith(func(i int) bool { return i%2 == 0 })

	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
}

// TestReverseMethod 测试 Query.Reverse 方法
func TestReverseMethod(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Reverse().ToSlice()

	expected := []int{5, 4, 3, 2, 1}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// TestSingle 测试获取单一元素
func TestSingle(t *testing.T) {
	single := []int{42}
	result := From(single).Single()

	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// 多个元素应该返回零值
	multiple := []int{1, 2, 3}
	result2 := From(multiple).Single()
	if result2 != 0 {
		t.Errorf("Expected 0 for multiple elements, got %d", result2)
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
		t.Errorf("Expected sum 15, got %d", sum)
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
		t.Errorf("Unexpected indices: %v", indices)
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
		t.Errorf("SumIntBy: expected 15, got %d", sumInt)
	}

	sumInt64 := From(items).SumInt64By(func(i Item) int64 { return int64(i.Value) })
	if sumInt64 != 15 {
		t.Errorf("SumInt64By: expected 15, got %d", sumInt64)
	}

	sumFloat := From(items).SumFloat64By(func(i Item) float64 { return float64(i.Value) })
	if sumFloat != 15.0 {
		t.Errorf("SumFloat64By: expected 15.0, got %f", sumFloat)
	}
}

// TestAvgAllTypes 测试各种类型的平均值
func TestAvgAllTypes(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	avgInt := From(nums).AvgIntBy(func(i int) int { return i })
	if avgInt != 30.0 {
		t.Errorf("AvgIntBy: expected 30.0, got %f", avgInt)
	}

	avgInt64 := From(nums).AvgInt64By(func(i int) int64 { return int64(i) })
	if avgInt64 != 30.0 {
		t.Errorf("AvgInt64By: expected 30.0, got %f", avgInt64)
	}

	avgFloat := From(nums).AvgBy(func(i int) float64 { return float64(i) })
	if avgFloat != 30.0 {
		t.Errorf("AvgBy: expected 30.0, got %f", avgFloat)
	}
}

// TestCount 测试计数
func TestCount(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	count := From(nums).Count()

	if count != 5 {
		t.Errorf("Expected 5, got %d", count)
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
		t.Errorf("Expected 5 items, got %d", len(result))
	}
}

// TestToMapMethod 测试 Query.ToMap 方法
func TestToMapMethod(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{{"Alice", 30}, {"Bob", 25}}

	result := From(people).ToMap(func(p Person) map[string]any {
		return map[string]any{"name": p.Name, "age": p.Age}
	})

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
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
			t.Errorf("Index %d: expected %s, got %s", i, expected[i], v)
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
		t.Errorf("Expected 3 distinct IDs, got %d", len(result))
	}
}

// TestRange 测试 Range 函数
func TestRange(t *testing.T) {
	result := Range(1, 5).ToSlice()

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// TestRepeat 测试 Repeat 函数
func TestRepeat(t *testing.T) {
	result := Repeat("hello", 3).ToSlice()

	if len(result) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(result))
	}
	for _, v := range result {
		if v != "hello" {
			t.Errorf("Expected 'hello', got '%s'", v)
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
		t.Errorf("Unexpected map: %v", result)
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
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestContains 测试包含
func TestContains(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}

	if !Contains(nums, 3) {
		t.Error("Expected true for 3")
	}
	if Contains(nums, 99) {
		t.Error("Expected false for 99")
	}
}

// TestIndexOfFunction 测试 IndexOf 函数
func TestIndexOfFunction(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	if IndexOf(nums, 30) != 2 {
		t.Error("Expected index 2 for 30")
	}
	if IndexOf(nums, 99) != -1 {
		t.Error("Expected -1 for 99")
	}
}

// TestLastIndexOf 测试 LastIndexOf 函数
func TestLastIndexOf(t *testing.T) {
	nums := []int{1, 2, 3, 2, 1}

	if LastIndexOf(nums, 2) != 3 {
		t.Error("Expected last index 3 for 2")
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
		t.Errorf("Expected same length, got %d", len(result))
	}

	// 验证原数组未被修改
	for i, v := range nums {
		if v != original[i] {
			t.Error("Original array was modified")
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
			t.Errorf("Index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// TestMinMax 测试最小最大值
func TestMinMax(t *testing.T) {
	if Min(3, 1, 4, 1, 5) != 1 {
		t.Error("Min failed")
	}
	if Max(3, 1, 4, 1, 5) != 5 {
		t.Error("Max failed")
	}

	// 空切片返回零值
	if Min[int]() != 0 {
		t.Error("Empty Min should return 0")
	}
}

// TestSumFunction 测试 Sum 函数
func TestSumFunction(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := Sum(nums)

	if result != 15 {
		t.Errorf("Expected 15, got %d", result)
	}

	floats := []float64{1.5, 2.5, 3.0}
	resultFloat := Sum(floats)
	if resultFloat != 7.0 {
		t.Errorf("Expected 7.0, got %f", resultFloat)
	}
}

// TestEverySomeNone 测试集合判断
func TestEverySomeNone(t *testing.T) {
	list := []int{1, 2, 3, 4, 5}

	// Every: 所有元素都在 list 中
	if !Every(list, []int{1, 3, 5}) {
		t.Error("Every failed")
	}
	if Every(list, []int{1, 6}) {
		t.Error("Every should return false for 6")
	}

	// Some: 至少一个元素在 list 中
	if !Some(list, []int{5, 6, 7}) {
		t.Error("Some failed")
	}
	if Some(list, []int{6, 7, 8}) {
		t.Error("Some should return false")
	}

	// None: 没有元素在 list 中
	if !None(list, []int{6, 7, 8}) {
		t.Error("None failed")
	}
	if None(list, []int{5, 6, 7}) {
		t.Error("None should return false for 5")
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
		t.Errorf("Left difference failed: %v", left)
	}
	if len(right) != 2 || right[0] != 6 || right[1] != 7 {
		t.Errorf("Right difference failed: %v", right)
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
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// 验证结果都在原数组中
	for _, v := range result {
		if !Contains(nums, v) {
			t.Errorf("Unexpected value: %d", v)
		}
	}
}

// ============================================================================
// 工具函数测试
// ============================================================================

// TestDefault 测试默认值
func TestDefault(t *testing.T) {
	if Default(0, 42) != 42 {
		t.Error("Default should return 42 for 0")
	}
	if Default(10, 42) != 10 {
		t.Error("Default should return 10 for non-zero")
	}
	if Default("", "default") != "default" {
		t.Error("Default should return 'default' for empty string")
	}
}

// TestEmpty 测试获取零值
func TestEmpty(t *testing.T) {
	if Empty[int]() != 0 {
		t.Error("Empty[int] should be 0")
	}
	if Empty[string]() != "" {
		t.Error("Empty[string] should be ''")
	}
}

// TestIsEmptyIsNotEmpty 测试空值判断
func TestIsEmptyIsNotEmpty(t *testing.T) {
	if !IsEmpty(0) {
		t.Error("IsEmpty(0) should be true")
	}
	if !IsEmpty("") {
		t.Error("IsEmpty('') should be true")
	}
	if IsEmpty(1) {
		t.Error("IsEmpty(1) should be false")
	}

	if IsNotEmpty(0) {
		t.Error("IsNotEmpty(0) should be false")
	}
	if !IsNotEmpty(1) {
		t.Error("IsNotEmpty(1) should be true")
	}
}

// TestTry 测试 Try 函数
func TestTry(t *testing.T) {
	// 成功的情况
	success := Try(func() error { return nil })
	if !success {
		t.Error("Try should return true for success")
	}

	// 失败的情况
	failure := Try(func() error { return fmt.Errorf("error") })
	if failure {
		t.Error("Try should return false for error")
	}

	// Panic 的情况
	panicCase := Try(func() error { panic("panic") })
	if panicCase {
		t.Error("Try should return false for panic")
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
		t.Error("Catch function should be called")
	}
}

// TestIF 测试三目运算
func TestIF(t *testing.T) {
	if IF(true, "yes", "no") != "yes" {
		t.Error("IF(true) should return 'yes'")
	}
	if IF(false, "yes", "no") != "no" {
		t.Error("IF(false) should return 'no'")
	}
	if IF(1 > 0, 100, 200) != 100 {
		t.Error("IF(1>0) should return 100")
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
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
}

// TestIntersectComparable 测试优化的交集
func TestIntersectComparable(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := IntersectComparable(From(nums1), From(nums2)).ToSlice()

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
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
		t.Errorf("Expected [1], got %v", result)
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
		t.Errorf("Expected 2 items, got %d: %v", len(result), result)
	}
}

// TestSkipMoreThanLength 测试 Skip 超过长度
func TestSkipMoreThanLength(t *testing.T) {
	nums := []int{1, 2, 3}
	result := From(nums).Skip(10).ToSlice()

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(result))
	}
}

// TestTakeMoreThanLength 测试 Take 超过长度
func TestTakeMoreThanLength(t *testing.T) {
	nums := []int{1, 2, 3}
	result := From(nums).Take(10).ToSlice()

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

// TestAnyWithNoMatch 测试 AnyWith 无匹配
func TestAnyWithNoMatch(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9}
	result := From(nums).AnyWith(func(i int) bool { return i%2 == 0 })

	if result {
		t.Error("Expected false, no even numbers")
	}
}

// TestFirstWithNoMatch 测试 FirstWith 无匹配
func TestFirstWithNoMatch(t *testing.T) {
	nums := []int{1, 3, 5, 7, 9}
	result := From(nums).FirstWith(func(i int) bool { return i > 100 })

	if result != 0 {
		t.Errorf("Expected 0 for no match, got %d", result)
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
		t.Errorf("Expected 3 iterations, got %d", count)
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
		t.Errorf("Expected last index 2, got %d", lastIdx)
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
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
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
		t.Error("Expected success after retries")
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestSumInt8By 测试 int8 求和
func TestSumInt8To64(t *testing.T) {
	type Num struct{ Val int }
	nums := []Num{{1}, {2}, {3}}

	sum8 := From(nums).SumInt8By(func(n Num) int8 { return int8(n.Val) })
	if sum8 != 6 {
		t.Errorf("SumInt8By: expected 6, got %d", sum8)
	}

	sum16 := From(nums).SumInt16By(func(n Num) int16 { return int16(n.Val) })
	if sum16 != 6 {
		t.Errorf("SumInt16By: expected 6, got %d", sum16)
	}

	sum32 := From(nums).SumInt32By(func(n Num) int32 { return int32(n.Val) })
	if sum32 != 6 {
		t.Errorf("SumInt32By: expected 6, got %d", sum32)
	}
}

// TestSumUIntTypes 测试无符号整数求和
func TestSumUIntTypes(t *testing.T) {
	type Num struct{ Val uint }
	nums := []Num{{1}, {2}, {3}}

	sumU := From(nums).SumUIntBy(func(n Num) uint { return n.Val })
	if sumU != 6 {
		t.Errorf("SumUIntBy: expected 6, got %d", sumU)
	}

	sumU8 := From(nums).SumUInt8By(func(n Num) uint8 { return uint8(n.Val) })
	if sumU8 != 6 {
		t.Errorf("SumUInt8By: expected 6, got %d", sumU8)
	}

	sumU16 := From(nums).SumUInt16By(func(n Num) uint16 { return uint16(n.Val) })
	if sumU16 != 6 {
		t.Errorf("SumUInt16By: expected 6, got %d", sumU16)
	}

	sumU32 := From(nums).SumUInt32By(func(n Num) uint32 { return uint32(n.Val) })
	if sumU32 != 6 {
		t.Errorf("SumUInt32By: expected 6, got %d", sumU32)
	}

	sumU64 := From(nums).SumUInt64By(func(n Num) uint64 { return uint64(n.Val) })
	if sumU64 != 6 {
		t.Errorf("SumUInt64By: expected 6, got %d", sumU64)
	}
}

// TestSumFloat32 测试 float32 求和
func TestSumFloat32(t *testing.T) {
	type Num struct{ Val float32 }
	nums := []Num{{1.5}, {2.5}, {3.0}}

	sum := From(nums).SumFloat32By(func(n Num) float32 { return n.Val })
	if sum != 7.0 {
		t.Errorf("Expected 7.0, got %f", sum)
	}
}
