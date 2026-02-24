package linq

// Count 返回序列中的元素个数
func (q Query[T]) Count() int {
	if q.fastSlice != nil && q.fastWhere == nil {
		return len(q.fastSlice)
	}
	count := 0
	for range q.iterate {
		count++
	}
	return count
}

// CountWith 统计满足条件的元素个数
func (q Query[T]) CountWith(predicate func(T) bool) int {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		count := 0
		for _, v := range source {
			if preFilter != nil && !preFilter(v) {
				continue
			}
			if predicate(v) {
				count++
			}
		}
		return count
	}
	count := 0
	for item := range q.iterate {
		if predicate(item) {
			count++
		}
	}
	return count
}

// Any 判断序列是否包含任何元素
func (q Query[T]) Any() bool {
	if q.fastSlice != nil {
		if q.fastWhere == nil {
			return len(q.fastSlice) > 0
		}
		for _, v := range q.fastSlice {
			if q.fastWhere(v) {
				return true
			}
		}
		return false
	}
	for range q.iterate {
		return true
	}
	return false
}

// AnyWith 判断序列是否包含满足指定条件的元素
func (q Query[T]) AnyWith(predicate func(T) bool) bool {
	for item := range q.iterate {
		if predicate(item) {
			return true
		}
	}
	return false
}

// All 判断序列中的所有元素是否都满足指定条件
func (q Query[T]) All(predicate func(T) bool) bool {
	for item := range q.iterate {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// Sum 计算数值序列的和
func Sum[T Integer | Float | Complex](q Query[T]) T {
	var sum T
	for item := range q.iterate {
		sum += item
	}
	return sum
}

// SumBy 根据选择器获取成员和
func SumBy[T any, R Integer | Float | Complex](q Query[T], selector func(T) R) R {
	var sum R
	for item := range q.iterate {
		sum += selector(item)
	}
	return sum
}

// Average 计算数值序列的平均值（float64）
func Average[T Integer | Float](q Query[T]) float64 {
	var sum float64
	count := 0
	for item := range q.iterate {
		sum += float64(item)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// AverageBy 根据选择器计算平均值
func AverageBy[T any, R Integer | Float](q Query[T], selector func(T) R) float64 {
	var sum float64
	count := 0
	for item := range q.iterate {
		sum += float64(selector(item))
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// Contains 判断序列中是否包含指定的元素
func Contains[T comparable](q Query[T], value T) bool {
	return q.AnyWith(func(t T) bool { return t == value })
}

// First 返回第一元素，如果没有则返回零值
func (q Query[T]) First() T {
	for item := range q.iterate {
		return item
	}
	var zero T
	return zero
}

// FirstWith 返回满足条件的第一个元素
func (q Query[T]) FirstWith(predicate func(T) bool) T {
	for item := range q.iterate {
		if predicate(item) {
			return item
		}
	}
	var zero T
	return zero
}

// Last 返回最后一个元素，如果没有则返回零值
func (q Query[T]) Last() T {
	if q.fastSlice != nil {
		source := q.fastSlice
		pre := q.fastWhere
		if pre == nil {
			if len(source) > 0 {
				return source[len(source)-1]
			}
		} else {
			for i := len(source) - 1; i >= 0; i-- {
				if pre(source[i]) {
					return source[i]
				}
			}
		}
	}
	var last T
	for item := range q.iterate {
		last = item
	}
	return last
}

// LastWith 返回满足条件的最后一个元素
func (q Query[T]) LastWith(predicate func(T) bool) T {
	var last T
	for item := range q.iterate {
		if predicate(item) {
			last = item
		}
	}
	return last
}

// FirstDefault 返回第一个元素，若空返回 defaultValue
func (q Query[T]) FirstDefault(defaultValue ...T) T {
	for item := range q.iterate {
		return item
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	var zero T
	return zero
}

// LastDefault 返回最后一个元素，若空返回 defaultValue
func (q Query[T]) LastDefault(defaultValue ...T) T {
	var last T
	found := false
	for item := range q.iterate {
		last = item
		found = true
	}
	if found {
		return last
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	var zero T
	return zero
}

// Single 返回包含且仅包含一个元素的序列的那个元素，如果不等于1个返回零值
func (q Query[T]) Single() T {
	var val T
	count := 0
	for item := range q.iterate {
		val = item
		count++
		if count > 1 {
			var zero T
			return zero
		}
	}
	if count == 0 {
		var zero T
		return zero
	}
	return val
}

// SingleWith 返回满足条件的那个元素，如果不等于1个返回零值
func (q Query[T]) SingleWith(predicate func(T) bool) T {
	return q.Where(predicate).Single()
}

// SingleDefault 返回包含且仅包含一个元素的序列的那个元素，如果不等于1个返回默认值或者零值
func (q Query[T]) SingleDefault(defaultValue ...T) T {
	var val T
	count := 0
	for item := range q.iterate {
		val = item
		count++
		if count > 1 {
			if len(defaultValue) > 0 {
				return defaultValue[0]
			}
			var zero T
			return zero
		}
	}
	if count == 0 {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		var zero T
		return zero
	}
	return val
}

// IndexOf 返回元素的索引
func IndexOf[T comparable](q Query[T], value T) int {
	index := 0
	for item := range q.iterate {
		if item == value {
			return index
		}
		index++
	}
	return -1
}

// IndexOfWith 返回满足条件的元素的索引
func (q Query[T]) IndexOfWith(predicate func(T) bool) int {
	index := 0
	for item := range q.iterate {
		if predicate(item) {
			return index
		}
		index++
	}
	return -1
}

// LastIndexOf 返回元素最后出现的索引
func LastIndexOf[T comparable](q Query[T], value T) int {
	index := 0
	last := -1
	for item := range q.iterate {
		if item == value {
			last = index
		}
		index++
	}
	return last
}

// LastIndexOfWith 返回满足条件的元素最后出现的索引
func (q Query[T]) LastIndexOfWith(predicate func(T) bool) int {
	index := 0
	last := -1
	for item := range q.iterate {
		if predicate(item) {
			last = index
		}
		index++
	}
	return last
}
