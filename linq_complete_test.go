// go test -v linq_complete_test.go linq.go

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
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestTake 测试获取前N个元素
func TestTake(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := From(nums).Take(3).ToSlice()

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
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
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
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
	result := Distinct(From(nums)).ToSlice()

	expected := []int{1, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestExcept 测试差集
func TestExcept(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := Except(From(nums1), From(nums2)).ToSlice()

	expected := []int{1, 2}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestIntersectMethod 测试 Query.Intersect 方法
func TestIntersectMethod(t *testing.T) {
	nums1 := []int{1, 2, 3, 4, 5}
	nums2 := []int{3, 4, 5, 6, 7}
	result := Intersect(From(nums1), From(nums2)).ToSlice()

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
	idx := From(nums).IndexOfWith(func(i int) bool { return i == 30 })

	if idx != 2 {
		t.Errorf("期望索引 2，实际得到 %d", idx)
	}

	// 不存在的元素
	idx2 := From(nums).IndexOfWith(func(i int) bool { return i == 99 })
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

	sumInt := SumBy(From(items), func(i Item) int { return i.Value })
	if sumInt != 15 {
		t.Errorf("SumIntBy: 期望 15，实际得到 %d", sumInt)
	}

	sumInt64 := SumBy(From(items), func(i Item) int64 { return int64(i.Value) })
	if sumInt64 != 15 {
		t.Errorf("SumInt64By: 期望 15，实际得到 %d", sumInt64)
	}

	sumFloat := SumBy(From(items), func(i Item) float64 { return float64(i.Value) })
	if sumFloat != 15.0 {
		t.Errorf("SumFloat64By: 期望 15.0，实际得到 %f", sumFloat)
	}
}

// TestAvgAllTypes 测试各种类型的平均值
func TestAvgAllTypes(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	avgInt := AverageBy(From(nums), func(i int) int { return i })
	if avgInt != 30.0 {
		t.Errorf("AvgIntBy: 期望 30.0，实际得到 %f", avgInt)
	}

	avgInt64 := AverageBy(From(nums), func(i int) int64 { return int64(i) })
	if avgInt64 != 30.0 {
		t.Errorf("AvgInt64By: 期望 30.0，实际得到 %f", avgInt64)
	}

	avgFloat := AverageBy(From(nums), func(i int) float64 { return float64(i) })
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

	ch := From(nums).ToChannel(context.Background())

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

	result := From(people).ToMapSlice(func(p Person) map[string]Person {
		return map[string]Person{p.Name: p}
	})

	if len(result) != 2 {
		t.Errorf("期望 2 个元素，实际得到 %d", len(result))
	}
	if result[0]["Alice"].Age != 30 {
		t.Errorf("期望 Alice 年龄 30，实际得到 %d", result[0]["Alice"].Age)
	}
}

// ============================================================================
// 独立函数测试
// ============================================================================

// TestSelect 测试 Select 函数
func TestSelect(t *testing.T) {
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

// TestDistinctSelect 测试 DistinctSelect 函数
func TestDistinctSelect(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items := []Item{{1, "a"}, {2, "b"}, {1, "c"}, {3, "d"}, {2, "e"}}

	result := DistinctSelect(From(items), func(i Item) int { return i.ID }).ToSlice()

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

	if !Contains(From(nums), 3) {
		t.Error("3 期望返回 true")
	}
	if Contains(From(nums), 99) {
		t.Error("99 期望返回 false")
	}
}

// TestIndexOfFunction 测试 IndexOf 函数
func TestIndexOfFunction(t *testing.T) {
	nums := []int{10, 20, 30, 40, 50}

	if IndexOf(From(nums), 30) != 2 {
		t.Error("30 期望索引 2")
	}
	if IndexOf(From(nums), 99) != -1 {
		t.Error("99 期望 -1")
	}
}

// TestLastIndexOf 测试 LastIndexOf 函数
func TestLastIndexOf(t *testing.T) {
	nums := []int{1, 2, 3, 2, 1}

	if LastIndexOf(From(nums), 2) != 3 {
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
	result := Sum(From(nums))

	if result != 15 {
		t.Errorf("期望 15，实际得到 %d", result)
	}

	floats := []float64{1.5, 2.5, 3.0}
	resultFloat := Sum(From(floats))
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
	result := SliceIntersect(list1, list2)

	expected := []int{3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// Difference 测试差异
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
	result := SliceUnion(list1, list2)

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestNoEmpty 测试过滤空值
func TestNoEmpty(t *testing.T) {
	strs := []string{"a", "", "b", "", "c"}
	result := WithoutEmpty(strs)

	expected := []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestGtZero 测试过滤大于零的值
func TestGtZero(t *testing.T) {
	nums := []int{-2, -1, 0, 1, 2, 3}
	result := WithoutLEZero(nums)

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
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
		if !Contains(From(nums), v) {
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
	if SliceEmpty[int]() != 0 {
		t.Error("Empty[int] 应该是 0")
	}
	if SliceEmpty[string]() != "" {
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
func TestSliceTry(t *testing.T) {
	// 成功的情况
	success := SliceTry(func() error { return nil })
	if !success {
		t.Error("Try 成功时应该返回 true")
	}

	// 失败的情况
	failure := SliceTry(func() error { return fmt.Errorf("error") })
	if failure {
		t.Error("Try 错误时应该返回 false")
	}

	// Panic 的情况
	panicCase := SliceTry(func() error { panic("panic") })
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

// TestOrdering 测试排序相关功能
func TestOrdering(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Alice", 30},
		{"Bob", 20},
		{"Charlie", 30},
		{"David", 20},
	}

	q := From(people)
	if q.HasOrder() {
		t.Error("初始 Query 不应该有排序规则")
	}

	// 按年龄升序，再按名字降序
	result := ThenByDescending(OrderBy(q, func(p Person) int { return p.Age }), func(p Person) string { return p.Name }).
		ToSlice()

	if result[0].Name != "David" || result[0].Age != 20 {
		t.Errorf("排序错误: %v", result)
	}
	if result[1].Name != "Bob" || result[1].Age != 20 {
		t.Errorf("排序错误: %v", result)
	}

	// 验证 HasOrder
	qOrdered := OrderBy(q, func(p Person) int { return p.Age })
	if !qOrdered.HasOrder() {
		t.Error("OrderBy 后的 Query 应该有排序规则")
	}

	// 测试 OrderByDescending 和 ThenBy
	result2 := ThenBy(OrderByDescending(q, func(p Person) int { return p.Age }), func(p Person) string { return p.Name }).
		ToSlice()

	if result2[0].Name != "Alice" || result2[0].Age != 30 {
		t.Errorf("排序错误: %v", result2)
	}
}

// TestGrouping 测试分组
func TestGrouping(t *testing.T) {
	type Person struct {
		Name string
		City string
	}
	people := []Person{
		{"Alice", "New York"},
		{"Bob", "Tokyo"},
		{"Charlie", "New York"},
	}

	// GroupBy
	groups := GroupBy(From(people), func(p Person) string { return p.City }).ToSlice()
	if len(groups) != 2 {
		t.Errorf("期望 2 个分组，实际得到 %d", len(groups))
	}

	// GroupBySelect
	groupsSelect := GroupBySelect(From(people),
		func(p Person) string { return p.City },
		func(p Person) string { return p.Name },
	).ToSlice()

	for _, g := range groupsSelect {
		if g.Key == "New York" {
			if len(g.Value) != 2 {
				t.Errorf("New York 分组应该有 2 人，实际得到 %d", len(g.Value))
			}
		}
	}
}

// TestPageComplete 测试分页
func TestPageComplete(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := From(nums).Page(2, 3).ToSlice() // 第2页，每页3个 -> 4, 5, 6

	expected := []int{4, 5, 6}
	if len(result) != 3 || result[0] != 4 || result[1] != 5 || result[2] != 6 {
		t.Errorf("期望 %v，实际得到 %v", expected, result)
	}
}

// TestParallelProcesses 测试并发处理
func TestParallelProcesses(t *testing.T) {
	nums := Range(1, 10)

	// SelectAsync
	results := SelectAsync(nums, 2, func(i int) int {
		return i * 10
	}).ToSlice()

	if len(results) != 10 {
		t.Errorf("SelectAsync 期望 10 个结果，实际得到 %d", len(results))
	}

	// ForEachParallel
	var sum atomic.Int64
	From([]int{1, 2, 3, 4, 5}).ForEachParallel(2, func(i int) {
		sum.Add(int64(i))
	})
	if sum.Load() != 15 {
		t.Errorf("ForEachParallel 期望总和 15，实际得到 %d", sum.Load())
	}
}

// TestWhereSelectComplete 测试 WhereSelect
func TestWhereSelectComplete(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	result := WhereSelect(From(nums), func(i int) (string, bool) {
		if i%2 == 0 {
			return fmt.Sprintf("even-%d", i), true
		}
		return "", false
	}).ToSlice()

	expected := []string{"even-2", "even-4"}
	if len(result) != 2 || result[0] != "even-2" || result[1] != "even-4" {
		t.Errorf("期望 %v，实际得到 %v", expected, result)
	}
}

// TestAppendToComplete 测试 AppendTo
func TestAppendToComplete(t *testing.T) {
	nums := []int{1, 2, 3}
	dest := []int{0}
	result := From(nums).AppendTo(dest)

	if len(result) != 4 || result[3] != 3 {
		t.Errorf("意外的结果: %v", result)
	}
}

// TestFirstLastDefault 测试默认值获取
func TestFirstLastDefault(t *testing.T) {
	empty := From([]int{})
	if empty.FirstDefault(99) != 99 {
		t.Error("FirstDefault 失败")
	}
	if empty.LastDefault(88) != 88 {
		t.Error("LastDefault 失败")
	}

	nums := From([]int{1, 2, 3})
	if nums.FirstDefault(99) != 1 {
		t.Error("FirstDefault 应该返回第一个元素")
	}
	if nums.LastDefault(99) != 3 {
		t.Error("LastDefault 应该返回最后一个元素")
	}

	// 非空但未提供默认值
	if nums.FirstDefault() != 1 {
		t.Error("FirstDefault(无默认值) 应该返回第一个元素")
	}
	if nums.LastDefault() != 3 {
		t.Error("LastDefault(无默认值) 应该返回最后一个元素")
	}

	// 首/末元素为零值时，不应误判为空
	zerosFirst := From([]int{0, 2, 3})
	if zerosFirst.FirstDefault(99) != 0 {
		t.Error("FirstDefault 应该返回 0 而不是默认值")
	}
	zerosLast := From([]int{1, 2, 0})
	if zerosLast.LastDefault(99) != 0 {
		t.Error("LastDefault 应该返回 0 而不是默认值")
	}

	// 过滤后首元素为零值
	filtered := From([]int{1, 0, 2}).Where(func(i int) bool { return i%2 == 0 })
	if filtered.FirstDefault(99) != 0 {
		t.Error("过滤后 FirstDefault 应该返回 0 而不是默认值")
	}
}

// TestStaticFunctions 测试独立工具函数
func TestStaticFunctions(t *testing.T) {
	nums := []int{1, 2, 3}

	// Map / MapIndexed
	m1 := Map(nums, func(i int) int { return i * 2 })
	if m1[0] != 2 {
		t.Error("Map 失败")
	}
	m2 := MapIndexed(nums, func(i int, idx int) int { return i + idx })
	if m2[1] != 3 {
		t.Error("MapIndexed 失败")
	}

	// Where / WhereIndexed
	w1 := Where(nums, func(i int) bool { return i > 1 })
	if len(w1) != 2 {
		t.Error("Where 失败")
	}
	w2 := WhereIndexed(nums, func(i int, idx int) bool { return idx == 0 })
	if len(w2) != 1 || w2[0] != 1 {
		t.Error("WhereIndexed 失败")
	}

	// Without / WithoutIndex
	wo := Without(nums, 2)
	if len(wo) != 2 || wo[1] != 3 {
		t.Error("Without 失败")
	}

	wi := WithoutIndex(nums, 1)
	if len(wi) != 2 || wi[1] != 3 {
		t.Error("WithoutIndex 失败")
	}

	// Equal / EqualBy
	if !Equal([]int{1, 2}, 1, 2) {
		t.Error("Equal 失败")
	}
	if !EqualBy([]int{1}, []int{2}, func(i int) int { return 0 }) {
		t.Error("EqualBy 失败")
	}

	// SliceContainsBy
	if !SliceContainsBy(nums, func(i int) bool { return i == 2 }) {
		t.Error("SliceContainsBy 失败")
	}

	// BigData Path (通过模拟大数据触发)
	bigList := make([]int, 2001)
	for i := range 2001 {
		bigList[i] = i
	}
	bigSubset := make([]int, 101)
	for i := range 101 {
		bigSubset[i] = i
	}
	Every(bigList, bigSubset) // 触发 EveryBigData
	Some(bigList, bigSubset)  // 触发 SomeBigData
	None(bigList, bigSubset)  // 触发 NoneBigData
}

// TestMinMaxByIndependent 测试 MinBy/MaxBy
func TestMinMaxByIndependent(t *testing.T) {
	nums := []int{10, 5, 20, 15}
	q := From(nums)

	if MinBy(q, func(i int) int { return i }) != 5 {
		t.Error("MinBy 失败")
	}
	if MaxBy(q, func(i int) int { return i }) != 20 {
		t.Error("MaxBy 失败")
	}
}

// TestUnionSelect 测试 并集 函数
func TestUnionSelect(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items1 := []Item{{1, "a"}, {2, "b"}, {3, "c"}}
	items2 := []Item{{2, "x"}, {3, "y"}}

	// UnionSelect 返回的是 selector 的结果类型 (int)
	result := UnionSelect(From(items1), From(items2), func(i Item) int { return i.ID }).ToSlice()

	if len(result) != 3 {
		t.Errorf("期望 3 个元素，实际得到 %v", result)
	}
}

// TestExceptSelect 测试 差集 函数
func TestExceptSelect(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items1 := []Item{{1, "a"}, {2, "b"}, {3, "c"}}
	items2 := []Item{{2, "x"}, {3, "y"}}

	// ExceptSelect 返回的是 selector 的结果类型 (int)
	result := ExceptSelect(From(items1), From(items2), func(i Item) int { return i.ID }).ToSlice()

	if len(result) != 1 || result[0] != 1 {
		t.Errorf("期望 [1]，实际得到 %v", result)
	}
}

// TestIntersectSelect 测试 交集 函数
func TestIntersectSelect(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	items1 := []Item{{1, "a"}, {2, "b"}, {3, "c"}}
	items2 := []Item{{2, "x"}, {3, "y"}}

	// IntersectSelect 返回的是 selector 的结果类型 (int)
	result := IntersectSelect(From(items1), From(items2), func(i Item) int { return i.ID }).ToSlice()

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
	result := Union(From(nums1), From(nums2)).ToSlice()

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}
}

// TestTryWithRetry 测试 Try 重试
func TestTryWithRetry(t *testing.T) {
	attempts := 0
	success := SliceTry(func() error {
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

// TestSumAllRemainingTypes 测试所有数值类型的求和
func TestSumAllRemainingTypes(t *testing.T) {
	type Num struct{ Val int }
	items := []Num{{1}, {2}, {3}}
	q := From(items)

	if SumBy(q, func(n Num) int8 { return int8(n.Val) }) != 6 {
		t.Error("SumInt8By 失败")
	}
	if SumBy(q, func(n Num) int16 { return int16(n.Val) }) != 6 {
		t.Error("SumInt16By 失败")
	}
	if SumBy(q, func(n Num) int32 { return int32(n.Val) }) != 6 {
		t.Error("SumInt32By 失败")
	}
	if SumBy(q, func(n Num) float32 { return float32(n.Val) }) != 6.0 {
		t.Error("SumFloat32By 失败")
	}

	type UNum struct{ Val uint }
	uitems := []UNum{{1}, {2}, {3}}
	uq := From(uitems)

	if SumBy(uq, func(n UNum) uint { return n.Val }) != 6 {
		t.Error("SumUIntBy 失败")
	}
	if SumBy(uq, func(n UNum) uint8 { return uint8(n.Val) }) != 6 {
		t.Error("SumUInt8By 失败")
	}
	if SumBy(uq, func(n UNum) uint16 { return uint16(n.Val) }) != 6 {
		t.Error("SumUInt16By 失败")
	}
	if SumBy(uq, func(n UNum) uint32 { return uint32(n.Val) }) != 6 {
		t.Error("SumUInt32By 失败")
	}
	if SumBy(uq, func(n UNum) uint64 { return uint64(n.Val) }) != 6 {
		t.Error("SumUInt64By 失败")
	}
}

// TestSelectAsyncCtx 测试带 Context 的并发 Select
func TestSelectAsyncCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 100
	nums := Range(0, count)

	result := SelectAsyncCtx(ctx, nums, 5, func(i int) int {
		return i * 2
	}).Take(10).ToSlice()

	cancel()

	if len(result) != 10 {
		t.Errorf("期望 10 个元素，实际得到 %d", len(result))
	}

	for i, v := range result {
		if v%2 != 0 {
			t.Errorf("索引 %d: 期望偶数，实际得到 %d", i, v)
		}
	}
}

// TestForEachParallelCtx 测试 ForEachParallelCtx
func TestForEachParallelCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nums := Range(0, 100)
	var processed atomic.Int32

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	nums.ForEachParallelCtx(ctx, 10, func(i int) {
		time.Sleep(100 * time.Millisecond)
		processed.Add(1)
	})

	if processed.Load() == 100 {
		t.Error("期望少于 100 个元素被处理")
	}
}

// TestEdgeCases 测试边缘情况
func TestEdgeCases(t *testing.T) {
	// EqualBy 不同长度
	if EqualBy([]int{1}, []int{1, 2}, func(i int) int { return i }) {
		t.Error("EqualBy 长度不同应返回 false")
	}
	if EqualBy([]int{1}, []int{2}, func(i int) int { return i }) {
		t.Error("EqualBy 值不同应返回 false")
	}

	// IndexOf fastPath with preFilter match/no-match
	q := From([]int{1, 2, 3, 4}).Where(func(i int) bool { return i > 2 }) // 结果集为 {3, 4}
	if q.IndexOfWith(func(i int) bool { return i == 4 }) != 1 {
		t.Errorf("Filtered IndexOf 失败, 得到 %d", q.IndexOfWith(func(i int) bool { return i == 4 }))
	}
	if q.IndexOfWith(func(i int) bool { return i == 1 }) != -1 {
		t.Error("Filtered IndexOf(hidden) 失败")
	}

	// Single with 0 items
	empty := From([]int{})
	if empty.Single() != 0 {
		t.Error("Empty Single 失败")
	}

	// All/Any with empty
	if !empty.All(func(i int) bool { return true }) {
		t.Error("Empty All 应为 true")
	}
	if empty.Any() {
		t.Error("Empty Any 应为 false")
	}
}

// TestToSlicePaths 测试 ToSlice 的不同路径
func TestToSlicePaths(t *testing.T) {
	nums := []int{1, 2, 3}
	// 直接路径 (copy)
	results := From(nums).ToSlice()
	if len(results) != 3 || results[0] != 1 {
		t.Error("Direct ToSlice 失败")
	}

	// 过滤路径 (fastSlice + fastWhere)
	results2 := From(nums).Where(func(i int) bool { return i > 1 }).ToSlice()
	if len(results2) != 2 || results2[0] != 2 {
		t.Error("Filtered ToSlice 失败")
	}

	// 延迟路径 (iterator)
	results3 := Range(1, 3).ToSlice()
	if len(results3) != 3 || results3[0] != 1 {
		t.Error("Lazy ToSlice 失败")
	}
}

// TestFromStringComplete 测试 FromString
func TestFromStringComplete(t *testing.T) {
	s := "hello世界"
	result := FromString(s).ToSlice()

	if len(result) != 7 {
		t.Errorf("期望 7 个字符，实际得到 %d: %v", len(result), result)
	}
	if result[5] != "世" || result[6] != "界" {
		t.Error("多字节字符解析错误")
	}
}

// TestTakeWhile 测试 TakeWhile
func TestTakeWhile(t *testing.T) {
	nums := []int{1, 2, 3, 4, 1, 2}
	result := From(nums).TakeWhile(func(i int) bool { return i < 4 }).ToSlice()

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("期望 %d 个元素，实际得到 %d", len(expected), len(result))
	}

	// 慢路径测试
	ch := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		ch <- i
	}
	close(ch)
	result2 := FromChannel(ch).TakeWhile(func(i int) bool { return i < 3 }).ToSlice()
	if len(result2) != 2 {
		t.Errorf("Lazy TakeWhile 失败: %v", result2)
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

	// 慢路径测试
	ch := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		ch <- i
	}
	close(ch)
	result2 := FromChannel(ch).SkipWhile(func(i int) bool { return i < 3 }).ToSlice()
	if len(result2) != 3 || result2[0] != 3 {
		t.Errorf("Lazy SkipWhile 失败: %v", result2)
	}
}

// TestLazyPaths 测试非切片源（慢路径）的覆盖
func TestLazyPaths(t *testing.T) {
	// 使用 Range 产生非切片 Query
	q := Range(1, 5) // 1, 2, 3, 4, 5

	// IndexOf
	if q.IndexOfWith(func(i int) bool { return i == 3 }) != 2 {
		t.Error("Lazy IndexOf 失败")
	}

	// All / Any / AnyWith
	if !q.All(func(i int) bool { return i > 0 }) {
		t.Error("Lazy All 失败")
	}
	if !q.Any() {
		t.Error("Lazy Any 失败")
	}
	if !q.AnyWith(func(i int) bool { return i == 5 }) {
		t.Error("Lazy AnyWith 失败")
	}

	// CountWith
	if q.CountWith(func(i int) bool { return i%2 == 0 }) != 2 {
		t.Error("Lazy CountWith 失败")
	}

	// First / FirstWith
	if q.First() != 1 {
		t.Error("Lazy First 失败")
	}
	if q.FirstWith(func(i int) bool { return i > 3 }) != 4 {
		t.Error("Lazy FirstWith 失败")
	}

	// Last / LastWith
	if q.Last() != 5 {
		t.Error("Lazy Last 失败")
	}
	if q.LastWith(func(i int) bool { return i < 3 }) != 2 {
		t.Error("Lazy LastWith 失败")
	}

	// ForEach / ForEachIndexed
	var count int
	q.ForEach(func(i int) bool {
		count++
		return true
	})
	if count != 5 {
		t.Error("Lazy ForEach 失败")
	}

	q.ForEachIndexed(func(idx int, val int) bool {
		if idx == 0 && val != 1 {
			t.Error("Lazy ForEachIndexed 失败")
		}
		return true
	})
}

// ============================================================================
// OrderedQuery 完整测试 (New Feature)
// ============================================================================

// TestOrderedQuery_Complete 测试 OrderedQuery 的排序和链式调用
func TestOrderedQuery_Complete(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		ID   int
	}
	// 数据准备: 乱序
	people := []Person{
		{"Alice", 30, 1},
		{"Bob", 20, 2}, // Age min
		{"Charlie", 30, 3},
		{"David", 20, 4},
		{"Eve", 25, 5},
	}

	// 1. Order (Asc) -> ID
	// 期望: 1, 2, 3, 4, 5 (已经是ID序? wait, source ID is 1,2,3,4,5 but mixed in list logic? No, list is defined above)
	// ID 顺序: 1, 2, 3, 4, 5.

	// 2. Order (Age Asc) -> Then (Name Desc)
	// Age: 20(Bob, David), 25(Eve), 30(Alice, Charlie)
	// Age=20 Group: Then Name Desc: David, Bob
	// Age=30 Group: Then Name Desc: Charlie, Alice
	// 期望顺序: David(20,4), Bob(20,2), Eve(25,5), Charlie(30,3), Alice(30,1)

	q := From(people).Order(Asc(func(p Person) int { return p.Age })).
		Then(Desc(func(p Person) string { return p.Name }))

	res := q.ToSlice()

	expectedIDs := []int{4, 2, 5, 3, 1}
	if len(res) != 5 {
		t.Fatalf("OrderedQuery 长度错误: %d", len(res))
	}
	for i, p := range res {
		if p.ID != expectedIDs[i] {
			t.Errorf("索引 %d 排序错误: 期望ID %d, 实际ID %d (%s, %d)", i, expectedIDs[i], p.ID, p.Name, p.Age)
		}
	}

	// 3. 测试 OrderByDescending (Age Desc) -> ThenBy (ID Asc)
	// Age Desc: 30(Alice, Charlie), 25(Eve), 20(Bob, David)
	// Age=30 Group: ID Asc -> Alice(1), Charlie(3)
	// Age=20 Group: ID Asc -> Bob(2), David(4)
	// 期望: Alice, Charlie, Eve, Bob, David (IDs: 1, 3, 5, 2, 4)

	q2 := From(people).Order(Desc(func(p Person) int { return p.Age })).
		Then(Asc(func(p Person) int { return p.ID }))
	res2 := q2.ToSlice()

	expectedIDs2 := []int{1, 3, 5, 2, 4}
	for i, p := range res2 {
		if p.ID != expectedIDs2[i] {
			t.Errorf("索引 %d 排序错误 (Desc): 期望ID %d, 实际ID %d", i, expectedIDs2[i], p.ID)
		}
	}
}

// TestOrderedQuery_Operations 测试 OrderedQuery 的操作方法 (Take, Skip, Where...)
func TestOrderedQuery_Operations(t *testing.T) {
	// 数据: [5, 1, 4, 2, 3] -> Ordered -> [1, 2, 3, 4, 5]
	nums := []int{5, 1, 4, 2, 3}
	q := From(nums).Order(Asc(func(i int) int { return i }))

	// Take
	takeRes := q.Take(2).ToSlice() // [1, 2]
	if len(takeRes) != 2 || takeRes[0] != 1 || takeRes[1] != 2 {
		t.Errorf("Ordered Take 失败: %v", takeRes)
	}

	// Skip
	skipRes := q.Skip(2).ToSlice() // [3, 4, 5]
	if len(skipRes) != 3 || skipRes[0] != 3 {
		t.Errorf("Ordered Skip 失败: %v", skipRes)
	}

	// Where (过滤偶数) -> [2, 4]
	whereRes := q.Where(func(i int) bool { return i%2 == 0 }).ToSlice()
	if len(whereRes) != 2 || whereRes[0] != 2 || whereRes[1] != 4 {
		t.Errorf("Ordered Where 失败: %v", whereRes)
	}

	// TakeWhile (< 3) -> [1, 2]
	twRes := q.TakeWhile(func(i int) bool { return i < 3 }).ToSlice()
	if len(twRes) != 2 || twRes[1] != 2 {
		t.Errorf("Ordered TakeWhile 失败: %v", twRes)
	}

	// SkipWhile (< 3) -> [3, 4, 5]
	swRes := q.SkipWhile(func(i int) bool { return i < 3 }).ToSlice()
	if len(swRes) != 3 || swRes[0] != 3 {
		t.Errorf("Ordered SkipWhile 失败: %v", swRes)
	}

	// Distinct (假设源有重复: [5, 1, 1, 2] -> Order -> [1, 1, 2, 5] -> Distinct -> [1, 2, 5])
	numsDup := []int{5, 1, 1, 2}
	qDup := From(numsDup).Order(Asc(func(i int) int { return i }))
	distinctRes := Distinct(qDup.ToQuery()).ToSlice()
	if len(distinctRes) != 3 || distinctRes[0] != 1 || distinctRes[1] != 2 || distinctRes[2] != 5 {
		t.Errorf("Ordered Distinct 失败: %v", distinctRes)
	}
}

// TestOrderedQuery_Traversal 测试 OrderedQuery 的遍历和聚合 (First, Last, ForEach)
func TestOrderedQuery_Traversal(t *testing.T) {
	// 数据: [3, 1, 2] -> Ordered -> [1, 2, 3]
	nums := []int{3, 1, 2}
	q := From(nums).Order(Asc(func(i int) int { return i }))

	// First
	if q.First() != 1 {
		t.Errorf("Ordered First 期望 1, 实际 %d", q.First())
	}

	// Last
	if q.Last() != 3 {
		t.Errorf("Ordered Last 期望 3, 实际 %d", q.Last())
	}

	// FirstDefault
	qEmpty := From([]int{}).Order(Asc(func(i int) int { return i }))
	if qEmpty.FirstDefault(99) != 99 {
		t.Errorf("Ordered FirstDefault 失败")
	}

	// IndexOf (找 2，位置索引应为 1)
	if idx := q.IndexOfWith(func(i int) bool { return i == 2 }); idx != 1 {
		t.Errorf("Ordered IndexOf 期望 1, 实际 %d", idx)
	}

	// ForEach
	var res []int
	q.ForEach(func(i int) bool {
		res = append(res, i)
		return true
	})
	if len(res) != 3 || res[0] != 1 || res[2] != 3 {
		t.Errorf("Ordered ForEach 顺序错误: %v", res)
	}

	// Reverse
	revRes := q.Reverse().ToSlice() // [3, 2, 1]
	if revRes[0] != 3 || revRes[2] != 1 {
		t.Errorf("Ordered Reverse 失败: %v", revRes)
	}
}
