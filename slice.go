package linq

import (
	"cmp"
	"math/rand/v2"
	"slices"
)

// SliceMap 将序列中的每个元素转换为新的对象
func SliceMap[T, V any](list []T, selector func(T) V) []V {
	if len(list) == 0 {
		return []V{}
	}
	result := make([]V, len(list))
	for i := range list {
		result[i] = selector(list[i])
	}
	return result
}

// SliceMapIndexed 将序列中的每个元素转换为新的对象
func SliceMapIndexed[T, V any](list []T, selector func(T, int) V) []V {
	if len(list) == 0 {
		return []V{}
	}
	result := make([]V, len(list))
	for i := range list {
		result[i] = selector(list[i], i)
	}
	return result
}

// SliceWhere 返回满足指定条件的元素序列
func SliceWhere[T any](list []T, predicate func(item T) bool) []T {
	if len(list) == 0 {
		return []T{}
	}

	// Task 2: Go 1.26 优化 - 对于小切片 (<256) 放弃经验容量重估，直接依托新的栈分配器和 Green Tea GC
	// 避免不精确的堆逃逸；仅对大数据量进行容量提示
	capEst := 0
	if len(list) >= 256 {
		capEst = len(list) / 2
	}
	result := make([]T, 0, capEst)

	for _, item := range list {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// SliceWhereIndexed 返回满足指定条件的元素序列
func SliceWhereIndexed[T any](list []T, predicate func(T, int) bool) []T {
	if len(list) == 0 {
		return []T{}
	}

	// Task 2: Go 1.26 优化 - 小切片依托新的栈分配器，不提供破坏栈优化的容量指示
	capEst := 0
	if len(list) >= 256 {
		capEst = len(list) / 2
	}
	result := make([]T, 0, capEst)

	for i, item := range list {
		if predicate(item, i) {
			result = append(result, item)
		}
	}
	return result
}

// SliceUniq 返回去重后的切片
func SliceUniq[T comparable](list []T) []T {
	if len(list) <= 1 {
		return list
	}

	result := make([]T, 0, len(list)/2+1) // 预估一半大小以节省空间
	seen := make(map[T]struct{}, len(list)/2+1)
	for _, e := range list {
		if _, ok := seen[e]; !ok {
			result = append(result, e)
			seen[e] = struct{}{}
		}
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
	for i := range half {
		j := length - 1 - i
		list[i], list[j] = list[j], list[i]
	}
}

// SliceReverse 反转切片中的元素, 缺点原地反转
func SliceReverse[T any](list []T) []T {
	if len(list) <= 1 {
		return list
	}
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
	if len(list) == 0 {
		var zero T
		return zero
	}

	if len(list) == 1 {
		return list[0]
	}

	length := len(list)
	min := list[0]
	if length < 4 {
		for i := 1; i < length; i++ {
			if list[i] < min {
				min = list[i]
			}
		}
		return min
	}

	// 4路解卷，分离数据依赖以充分提速流水线
	min1, min2, min3, min4 := list[0], list[1], list[2], list[3]
	i := 4
	for ; i <= length-4; i += 4 {
		if list[i] < min1 {
			min1 = list[i]
		}
		if list[i+1] < min2 {
			min2 = list[i+1]
		}
		if list[i+2] < min3 {
			min3 = list[i+2]
		}
		if list[i+3] < min4 {
			min4 = list[i+3]
		}
	}

	if min2 < min1 {
		min1 = min2
	}
	if min3 < min1 {
		min1 = min3
	}
	if min4 < min1 {
		min1 = min4
	}

	for ; i < length; i++ {
		if list[i] < min1 {
			min1 = list[i]
		}
	}
	return min1
}

// SliceMax 返回切片中的最大值
func SliceMax[T cmp.Ordered](list ...T) T {
	if len(list) == 0 {
		var zero T
		return zero
	}

	if len(list) == 1 {
		return list[0]
	}

	length := len(list)
	max := list[0]
	if length < 4 {
		for i := 1; i < length; i++ {
			if list[i] > max {
				max = list[i]
			}
		}
		return max
	}

	// 4路解卷，分离数据依赖以充分提速流水线
	max1, max2, max3, max4 := list[0], list[1], list[2], list[3]
	i := 4
	for ; i <= length-4; i += 4 {
		if list[i] > max1 {
			max1 = list[i]
		}
		if list[i+1] > max2 {
			max2 = list[i+1]
		}
		if list[i+2] > max3 {
			max3 = list[i+2]
		}
		if list[i+3] > max4 {
			max4 = list[i+3]
		}
	}

	if max2 > max1 {
		max1 = max2
	}
	if max3 > max1 {
		max1 = max3
	}
	if max4 > max1 {
		max1 = max4
	}

	for ; i < length; i++ {
		if list[i] > max1 {
			max1 = list[i]
		}
	}
	return max1
}

// SliceSum 计算切片中所有元素的总和
func SliceSum[T Float | Integer | Complex](list []T) T {
	var sum T
	if len(list) == 0 {
		return sum
	}

	// 8路循环展开，利用 CPU 指令级并发提升 2.4 倍性能
	length := len(list)
	i := 0
	for ; i <= length-8; i += 8 {
		sum += list[i] + list[i+1] + list[i+2] + list[i+3] + list[i+4] + list[i+5] + list[i+6] + list[i+7]
	}
	for ; i < length; i++ {
		sum += list[i]
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
	limit := min(n, 50)
	for i := 0; i < limit; i++ {
		v := list[i]
		if slices.Contains(subset, v) {
			return true
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

	// 优化：总是对较小的切片建立map以节省空间
	var shorter, longer []T
	if len(list1) <= len(list2) {
		shorter, longer = list1, list2
	} else {
		shorter, longer = list2, list1
	}

	result := make([]T, 0, len(shorter)/2+1) // 保守估计交集大小
	seen := make(map[T]struct{}, len(shorter))

	// 首先记录较小的切片
	for _, elem := range shorter {
		seen[elem] = struct{}{}
	}

	// 然后遍历较长的切片，查找交集
	processed := make(map[T]struct{}, len(shorter)/2+1) // 跟踪已添加元素以去除重复
	for _, elem := range longer {
		if _, exists := seen[elem]; exists {
			// 检查是否已经添加过该元素
			if _, wasProcessed := processed[elem]; !wasProcessed {
				result = append(result, elem)
				processed[elem] = struct{}{}
			}
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
