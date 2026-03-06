package linq

import (
	"cmp"
	"math/rand/v2"
	"slices"
)

// SliceMap 将序列中的每个元素转换为新的对象
func SliceMap[T, V any](list []T, selector func(T) V) []V {
	return SliceMapIndexed(list, func(item T, _ int) V { return selector(item) })
}

// SliceMapIndexed 将序列中的每个元素转换为新的对象
func SliceMapIndexed[T, V any](list []T, selector func(T, int) V) []V {
	result := make([]V, len(list))
	for i := range list {
		result[i] = selector(list[i], i)
	}
	return result
}

// SliceWhere 返回满足指定条件的元素序列
func SliceWhere[T any](list []T, predicate func(item T) bool) []T {
	return SliceWhereIndexed(list, func(item T, _ int) bool { return predicate(item) })
}

// SliceWhereIndexed 返回满足指定条件的元素序列
func SliceWhereIndexed[T any](list []T, predicate func(T, int) bool) []T {
	result := make([]T, 0, len(list))
	for i := range list {
		if predicate(list[i], i) {
			result = append(result, list[i])
		}
	}
	return result
}

// SliceUniq 返回去重后的切片
func SliceUniq[T comparable](list []T) []T {
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

// SliceContains 判断切片是否包含指定元素
func SliceContains[T comparable](list []T, element T) bool {
	return slices.Contains(list, element)
}

// SliceContainsBy 判断切片是否包含指定元素, 并附带条件
func SliceContainsBy[T any](list []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(list, predicate)
}

// SliceIndexOf 返回元素在切片中的索引，未找到返回 -1
func SliceIndexOf[T comparable](list []T, element T) int {
	for i, item := range list {
		if item == element {
			return i
		}
	}
	return -1
}

// SliceLastIndexOf 返回元素在切片中最后一次出现的索引，未找到返回 -1
func SliceLastIndexOf[T comparable](list []T, element T) int {
	length := len(list)
	for i := length - 1; i >= 0; i-- {
		if list[i] == element {
			return i
		}
	}
	return -1
}

func sliceReverse[T any](list []T) {
	length := len(list)
	half := length / 2
	for i := 0; i < half; i++ {
		j := length - 1 - i
		list[i], list[j] = list[j], list[i]
	}
}

// SliceReverse 反转切片中的元素, 缺点原地反转
func SliceReverse[T any](list []T) []T {
	sliceReverse(list)
	return list
}

// SliceCloneReverse 反转切片中的元素, 返回新的切片
func SliceCloneReverse[T any](list []T) []T {
	data := make([]T, len(list))
	copy(data, list)
	sliceReverse(data)
	return data
}

// SliceMin 返回切片中的最小值
func SliceMin[T cmp.Ordered](list ...T) T {
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

// SliceMax 返回切片中的最大值
func SliceMax[T cmp.Ordered](list ...T) T {
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

// SliceSum 计算切片中所有元素的总和
func SliceSum[T Float | Integer | Complex](list []T) T {
	var sum T = 0
	for _, val := range list {
		sum += val
	}
	return sum
}

// SliceEvery 判断子集中的所有元素都包含在集合中
func SliceEvery[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	// 子集极大 (M > 100) -> 选哈希
	// 或者list 极大且子集不极小 (N > 2000, M > 50) -> 选哈希
	if m > 100 || (n > 2000 && m > 50) {
		return SliceEveryBigData(list, subset)
	}
	// 小规模数据 (NM < 10000) -> 选线性 (无内存分配)
	return SliceEverySmallData(list, subset)
}

// SliceEverySmallData 判断子集中的所有元素都包含在集合中 适用于少数据
func SliceEverySmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if !SliceContains(list, subset[i]) {
			return false
		}
	}
	return true
}

// SliceEveryBigData 判断子集中的所有元素都包含在集合中 适用于大数据
func SliceEveryBigData[T comparable](list []T, subset []T) bool {
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

// SliceSome 判断集合中包含子集中的至少有一个元素 适用于少数据
func SliceSome[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	if n == 0 || m == 0 {
		return false
	}

	// 小数据优先线性扫描，避免建表开销
	if n < 128 || m < 128 {
		if n < m {
			for _, v := range list {
				if SliceContains(subset, v) {
					return true
				}
			}
			return false
		}
		for _, v := range subset {
			if SliceContains(list, v) {
				return true
			}
		}
		return false
	}

	// 投机命中：先用 list 的前一小段与 subset 做扫描，提升高命中场景性能
	limit := n
	if limit > 50 {
		limit = 50
	}
	for i := 0; i < limit; i++ {
		v := list[i]
		for _, s := range subset {
			if v == s {
				return true
			}
		}
	}

	// 回退：大数据对较小集合建表
	if n < m {
		seen := make(map[T]struct{}, n)
		for _, v := range list {
			seen[v] = struct{}{}
		}
		for _, v := range subset {
			if _, ok := seen[v]; ok {
				return true
			}
		}
		return false
	}

	seen := make(map[T]struct{}, m)
	for _, v := range subset {
		seen[v] = struct{}{}
	}
	for i := limit; i < n; i++ {
		if _, ok := seen[list[i]]; ok {
			return true
		}
	}
	return false
}

// SliceNone 判断集合中不包含子集的任何元素
func SliceNone[T comparable](list, subset []T) bool {
	return !SliceSome(list, subset)
}

// Intersect 返回两个切片的交集
func SliceIntersect[T comparable](list1 []T, list2 []T) []T {
	if len(list1) == 0 || len(list2) == 0 {
		return []T{}
	}
	capHint := len(list1)
	if len(list2) < capHint {
		capHint = len(list2)
	}
	result := make([]T, 0, capHint)
	// 0: 不存在 1: 存在于 list1 2: 已输出
	seen := make(map[T]uint8, len(list1))
	for _, elem := range list1 {
		seen[elem] = 1
	}
	for _, elem := range list2 {
		if seen[elem] == 1 {
			seen[elem] = 2
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

// SliceDifference 返回两个集合之间的差异, left返回的是list2中不存在的元素的集合, right返回的是list1中不存在的元素的集合
func SliceDifference[T comparable](list1, list2 []T) (left, right []T) {
	seenLeft := make(map[T]struct{}, len(list1))
	seenRight := make(map[T]struct{}, len(list2))
	left = make([]T, 0, len(list1))
	right = make([]T, 0, len(list2))
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

// SliceWithout 从切片中移除指定的元素
func SliceWithout[T comparable](list []T, exclude ...T) []T {
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

// SliceWithoutIndex 从切片中移除指定的索引的元素
func SliceWithoutIndex[T any](list []T, index ...int) []T {
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

// SliceWithoutEmpty 移除切片中的空值（零值）
func SliceWithoutEmpty[T comparable](list []T) []T {
	var empty T
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e != empty {
			result = append(result, e)
		}
	}
	return result
}

// SliceWithoutLEZero 移除切片中小于等于0 的值
func SliceWithoutLEZero[T Float | Integer](list []T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e > 0 {
			result = append(result, e)
		}
	}
	return result
}

// SliceEqual 比较两个列表是否相同
func SliceEqual[T comparable](list1 []T, list2 ...T) bool {
	return SliceEqualBy(list1, list2, func(item T) T { return item })
}

// SliceEqualBy 比较两个列表是否相同
func SliceEqualBy[T, K comparable](list1, list2 []T, selector func(T) K) bool {
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

// SliceRand 随机从切片中选取 count 个元素
func SliceRand[T any](list []T, count int) []T {
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

// SliceShuffle 随机打乱切片中的元素，返回新切片，原切片不变
func SliceShuffle[T any](list []T) []T {
	result := make([]T, len(list))
	copy(result, list)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// SliceConcat 合并多个结果集
func SliceConcat[T any](lists ...[]T) []T {
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
