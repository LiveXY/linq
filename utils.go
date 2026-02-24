package linq

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"time"
)

func Map[T, V comparable](list []T, selector func(T) V) []V {
	return MapIndexed(list, func(item T, _ int) V { return selector(item) })
}

// MapIndexed 将序列中的每个元素转换为新的对象
func MapIndexed[T, V comparable](list []T, selector func(T, int) V) []V {
	result := make([]V, len(list))
	for i := range list {
		result[i] = selector(list[i], i)
	}
	return result
}

// Where 返回满足指定条件的元素序列
func Where[T comparable](list []T, predicate func(item T) bool) []T {
	return WhereIndexed(list, func(item T, _ int) bool { return predicate(item) })
}

// Where 返回满足指定条件的元素序列
func WhereIndexed[T comparable](list []T, predicate func(T, int) bool) []T {
	result := make([]T, 0, len(list))
	for i := range list {
		if predicate(list[i], i) {
			result = append(result, list[i])
		}
	}
	return result
}

// Uniq 返回去重后的切片
func Uniq[T comparable](list []T) []T {
	result := []T{}
	seen := map[T]struct{}{}
	for _, e := range list {
		if _, ok := seen[e]; ok {
			continue
		}
		result = append(result, e)
		seen[e] = struct{}{}
	}
	return result
}

// Contains 判断切片是否包含指定元素
func SliceContains[T comparable](list []T, element T) bool {
	return slices.Contains(list, element)
}

// ContainsBy 判断切片是否包含指定元素, 并附带条件
func SliceContainsBy[T any](list []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(list, predicate)
}

// IndexOf 返回元素在切片中的索引，未找到返回 -1
func SliceIndexOf[T comparable](list []T, element T) int {
	for i, item := range list {
		if item == element {
			return i
		}
	}
	return -1
}

// LastIndexOf 返回元素在切片中最后一次出现的索引，未找到返回 -1
func SliceLastIndexOf[T comparable](list []T, element T) int {
	length := len(list)
	for i := length - 1; i >= 0; i-- {
		if list[i] == element {
			return i
		}
	}
	return -1
}

func reverse[T comparable](list []T) {
	length := len(list)
	half := length / 2
	for i := 0; i < half; i++ {
		j := length - 1 - i
		list[i], list[j] = list[j], list[i]
	}
}

// Reverse 反转切片中的元素, 缺点原地反转
func Reverse[T comparable](list []T) []T {
	reverse(list)
	return list
}

// CloneReverse 反转切片中的元素, 返回新的切片
func CloneReverse[T comparable](list []T) []T {
	data := make([]T, len(list))
	copy(data, list)
	reverse(data)
	return data
}

// Min 返回切片中的最小值
func Min[T cmp.Ordered](list ...T) T {
	var min T
	if len(list) == 0 {
		return min
	}
	min = list[0]
	for i := 1; i < len(list); i++ {
		item := list[i]
		if item < min {
			min = item
		}
	}
	return min
}

// Max 返回切片中的最大值
func Max[T cmp.Ordered](list ...T) T {
	var max T
	if len(list) == 0 {
		return max
	}
	max = list[0]
	for i := 1; i < len(list); i++ {
		item := list[i]
		if item > max {
			max = item
		}
	}
	return max
}

// MinBy 根据选择器返回的值计算最小值

// MaxBy 根据选择器返回的值计算最大值

// SumBy 根据选择器返回的值计算总和

// AvgBy 计算平均值，兼容所有类型

// Sum 计算切片中所有元素的总和
func SliceSum[T Float | Integer | Complex](list []T) T {
	var sum T = 0
	for _, val := range list {
		sum += val
	}
	return sum
}

// Every 判断子集中的所有元素都包含在集合中
func Every[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	// 子集极大 (M > 100) -> 选哈希
	// 或者list 极大且子集不极小 (N > 2000, M > 50) -> 选哈希
	if m > 100 || n > 2000 && m > 50 {
		return EveryBigData(list, subset)
	}
	// 小规模数据 (NM < 10000) -> 选线性 (无内存分配)
	return EverySmallData(list, subset)
}

// Every 判断子集中的所有元素都包含在集合中 适用于少数据
func EverySmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if !SliceContains(list, subset[i]) {
			return false
		}
	}
	return true
}

// Every 判断子集中的所有元素都包含在集合中 适用于大数据
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

// Some 判断集合中包含子集中的至少有一个元素 适用于少数据
func Some[T comparable](list, subset []T) bool {
	for i := range subset {
		if SliceContains(list, subset[i]) {
			return true
		}
	}
	return false
}

// None 判断集合中不包含子集的任何元素
func None[T comparable](list, subset []T) bool {
	for i := range subset {
		if SliceContains(list, subset[i]) {
			return false
		}
	}
	return true
}

// Intersect 返回两个切片的交集
func SliceIntersect[T comparable](list1 []T, list2 []T) []T {
	result := []T{}
	seen := map[T]struct{}{}
	for _, elem := range list1 {
		seen[elem] = struct{}{}
	}
	for _, elem := range list2 {
		if _, ok := seen[elem]; ok {
			result = append(result, elem)
		}
	}
	return result
}

// Union 返回两个切片的并集，自动去重
func SliceUnion[T comparable](lists ...[]T) []T {
	var capLen int
	for _, list := range lists {
		capLen += len(list)
	}
	result := make([]T, 0, capLen)
	seen := make(map[T]struct{}, capLen)
	for i := range lists {
		for j := range lists[i] {
			if _, ok := seen[lists[i][j]]; !ok {
				seen[lists[i][j]] = struct{}{}
				result = append(result, lists[i][j])
			}
		}
	}
	return result
}

// Difference 返回两个集合之间的差异, left返回的是list2中不存在的元素的集合, right返回的是list1中不存在的元素的集合
func Difference[T comparable](list1, list2 []T) (left, right []T) {
	seenLeft := map[T]struct{}{}
	seenRight := map[T]struct{}{}
	for i := range list1 {
		seenLeft[list1[i]] = struct{}{}
	}
	for i := range list2 {
		seenRight[list2[i]] = struct{}{}
	}
	for i := range list1 {
		if _, ok := seenRight[list1[i]]; !ok {
			left = append(left, list1[i])
		}
	}
	for i := range list2 {
		if _, ok := seenLeft[list2[i]]; !ok {
			right = append(right, list2[i])
		}
	}
	return left, right
}

// Without 从切片中移除指定的元素
func Without[T comparable](list []T, exclude ...T) []T {
	if len(exclude) == 0 || len(list) == 0 {
		return list
	}
	excludeSet := make(map[T]struct{}, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = struct{}{}
	}
	result := make([]T, 0, len(list))
	for _, e := range list {
		if _, ok := excludeSet[e]; !ok {
			result = append(result, e)
		}
	}
	return result
}

// WithoutIndex 从切片中移除指定的索引的元素
func WithoutIndex[T comparable](list []T, index ...int) []T {
	length := len(list)
	if len(index) == 0 || length == 0 {
		return list
	}
	removeSet := make(map[int]struct{}, len(index))
	for i := range index {
		if index[i] >= 0 && index[i] <= length-1 {
			removeSet[index[i]] = struct{}{}
		}
	}
	result := make([]T, 0, len(list))
	for i := range list {
		if _, ok := removeSet[i]; !ok {
			result = append(result, list[i])
		}
	}
	return result
}

// WithoutEmpty 移除切片中的空值（零值）
func WithoutEmpty[T comparable](list []T) []T {
	var empty T
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e != empty {
			result = append(result, e)
		}
	}
	return result
}

// WithoutLEZero 移除切片中小于等于0 的值
func WithoutLEZero[T Float | Integer](list []T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e > 0 {
			result = append(result, e)
		}
	}
	return result
}

// 比较两个列表是否相同
func Equal[T comparable](list1 []T, list2 ...T) bool {
	return EqualBy(list1, list2, func(item T) T { return item })
}

// 比较两个列表是否相同
func EqualBy[T, K comparable](list1, list2 []T, selector func(T) K) bool {
	if len(list1) != len(list2) {
		return false
	}
	if len(list1) == 0 {
		return true
	}
	counters := make(map[K]int, len(list1))
	for _, el := range list1 {
		counters[selector(el)]++
	}
	for _, el := range list2 {
		counters[selector(el)]--
	}
	for _, count := range counters {
		if count != 0 {
			return false
		}
	}
	return true
}

// Rand 随机从切片中选取 count 个元素
func Rand[T comparable](list []T, count int) []T {
	size := len(list)
	if count > size {
		count = size
	}
	if count <= 0 {
		return []T{}
	}
	templist := make([]T, size)
	copy(templist, list)
	results := make([]T, 0, count)
	for i := 0; i < count; i++ {
		remaining := size - i
		index := rand.IntN(remaining)
		results = append(results, templist[index])
		templist[index] = templist[remaining-1]
	}
	return results
}

// Shuffle 随机打乱切片中的元素，返回新切片，原切片不变
func Shuffle[T comparable](list []T) []T {
	result := make([]T, len(list))
	copy(result, list)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Default 如果值为空（零值），返回默认值
func Default[T comparable](v T, d ...T) T {
	if len(d) == 0 {
		return SliceEmpty[T]()
	}
	if IsEmpty(v) {
		return d[0]
	}
	return v
}

// Empty 返回类型的零值
func SliceEmpty[T comparable]() T {
	var zero T
	return zero
}

// IsEmpty 判断值是否为空（零值）
func IsEmpty[T comparable](v T) bool {
	var zero T
	return zero == v
}

// IsNotEmpty 判断值是否不为空（非零值）
func IsNotEmpty[T comparable](v T) bool {
	var zero T
	return zero != v
}

// Try 尝试执行函数，支持重试和延迟
func SliceTry(callback func() error, nums ...int) bool {
	num, second := 1, 0
	if len(nums) > 0 {
		num = nums[0]
	}
	if len(nums) > 1 {
		second = nums[1]
	}
	var i int
	for i < num {
		if try(callback) {
			return true
		}
		if second > 0 {
			time.Sleep(time.Duration(second) * time.Second)
		}
		i++
	}
	return false
}

// TryCatch 尝试执行函数，如果 panic 则执行 catch 函数
func TryCatch(callback func() error, catch func()) {
	if !try(callback) {
		catch()
	}
}
func try(callback func() error) (ok bool) {
	ok = true
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	err := callback()
	if err != nil {
		ok = false
	}
	return
}

// IF 三目运算
func IF[T comparable](cond bool, suc, fail T) T {
	if cond {
		return suc
	} else {
		return fail
	}
}

// Concat 合并多个结果集
func Concat[T comparable](lists ...[]T) []T {
	totalLen := 0
	for i := range lists {
		totalLen += len(lists[i])
	}
	result := make([]T, 0, totalLen)
	for i := range lists {
		result = append(result, lists[i]...)
	}
	return result
}
