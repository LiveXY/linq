package linq

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"
	"unicode/utf8"
)

// From 从切片创建 Query 查询对象
func From[T comparable](source []T) Query[T] {
	return Query[T]{
		iterate:   slices.Values(source),
		fastSlice: source,
		capacity:  len(source),
	}
}

// FromChannel 从只读 Channel 创建 Query 查询对象
func FromChannel[T comparable](source <-chan T) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for item := range source {
				if !yield(item) {
					break
				}
			}
		},
	}
}

// FromString 从字符串创建 Query 查询对象，每个元素为一个 UTF-8 字符
func FromString(source string) Query[string] {
	return Query[string]{
		iterate: func(yield func(string) bool) {
			pos := 0
			length := len(source)
			for pos < length {
				r, w := utf8.DecodeRuneInString(source[pos:])
				var item string
				if r == utf8.RuneError && w == 1 {
					item = string(r)
				} else {
					item = source[pos : pos+w]
				}
				pos += w
				if !yield(item) {
					break
				}
			}
		},
		capacity: len(source),
	}
}

// FromMap 从 Map 创建 Query 查询对象，每个元素为 KV 键值对
func FromMap[K, V comparable](source map[K]V) Query[KV[K, V]] {
	return Query[KV[K, V]]{
		iterate: func(yield func(KV[K, V]) bool) {
			for k, v := range maps.All(source) {
				if !yield(KV[K, V]{Key: k, Value: v}) {
					break
				}
			}
		},
		capacity: len(source),
	}
}

// QueryEmpty 创建一个空的 Query 查询对象
func QueryEmpty[T comparable]() Query[T] {
	return From([]T{})
}

// QueryRange 创建一个包含指定范围内整数序列的 Query 查询对象
func QueryRange(start, count int) Query[int] {
	if count <= 0 {
		return QueryEmpty[int]()
	}
	return Query[int]{
		iterate: func(yield func(int) bool) {
			end := start + count
			for i := start; i < end; i++ {
				if !yield(i) {
					return
				}
			}
		},
		capacity: count,
	}
}

// QueryRepeat 创建一个包含重复元素的 Query 查询对象
func QueryRepeat[T comparable](element T, count int) Query[T] {
	if count <= 0 {
		return QueryEmpty[T]()
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for i := 0; i < count; i++ {
				if !yield(element) {
					return
				}
			}
		},
		capacity: count,
	}
}

// QueryMinBy 根据选择器返回的值计算最小值
func QueryMinBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) (r V) {
	first := true
	for item := range q.iterate {
		n := selector(item)
		if first {
			r = n
			first = false
		} else if n < r {
			r = n
		}
	}
	return
}

// QueryMaxBy 根据选择器返回的值计算最大值
func QueryMaxBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) (r V) {
	first := true
	for item := range q.iterate {
		n := selector(item)
		if first {
			r = n
			first = false
		} else if n > r {
			r = n
		}
	}
	return
}

// QuerySumBy 根据选择器返回的值计算总和
func QuerySumBy[T comparable, V Integer | Float | Complex](q Query[T], selector func(T) V) (r V) {
	for item := range q.iterate {
		r += selector(item)
	}
	return
}

// QueryAvgBy 计算平均值，兼容所有类型
func QueryAvgBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) float64 {
	var sum float64
	var n int
	for item := range q.iterate {
		sum += float64(selector(item))
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// MinBy 根据选择器返回最小值
func MinBy[T comparable, R cmp.Ordered](q Query[T], selector func(T) R) T {
	if q.fastSlice != nil {
		var min T
		var minR R
		first := true
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			val := selector(v)
			if first || cmp.Compare(val, minR) < 0 {
				min = v
				minR = val
				first = false
			}
		}
		return min
	}
	var min T
	var minR R
	first := true
	for item := range q.iterate {
		val := selector(item)
		if first || cmp.Compare(val, minR) < 0 {
			min = item
			minR = val
			first = false
		}
	}
	return min
}

// MaxBy 根据选择器返回最大值
func MaxBy[T comparable, R cmp.Ordered](q Query[T], selector func(T) R) T {
	if q.fastSlice != nil {
		var max T
		var maxR R
		first := true
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			val := selector(v)
			if first || cmp.Compare(val, maxR) > 0 {
				max = v
				maxR = val
				first = false
			}
		}
		return max
	}
	var max T
	var maxR R
	first := true
	for item := range q.iterate {
		val := selector(item)
		if first || cmp.Compare(val, maxR) > 0 {
			max = item
			maxR = val
			first = false
		}
	}
	return max
}

// Sum 计算数值序列的和
func Sum[T Integer | Float | Complex](q Query[T]) T {
	if q.fastSlice != nil {
		var sum T
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			sum += v
		}
		return sum
	}
	var sum T
	for item := range q.iterate {
		sum += item
	}
	return sum
}

// SumBy 根据选择器获取成员和
func SumBy[T comparable, R Integer | Float | Complex](q Query[T], selector func(T) R) R {
	if q.fastSlice != nil {
		var sum R
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			sum += selector(v)
		}
		return sum
	}
	var sum R
	for item := range q.iterate {
		sum += selector(item)
	}
	return sum
}

// Average 计算数值序列的平均值（float64）
func Average[T Integer | Float](q Query[T]) float64 {
	if q.fastSlice != nil {
		var sum float64
		count := 0
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			sum += float64(v)
			count++
		}
		if count == 0 {
			return 0
		}
		return sum / float64(count)
	}
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
func AverageBy[T comparable, R Integer | Float](q Query[T], selector func(T) R) float64 {
	if q.fastSlice != nil {
		var sum float64
		count := 0
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			sum += float64(selector(v))
			count++
		}
		if count == 0 {
			return 0
		}
		return sum / float64(count)
	}
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

// AvgBy 顶级函数别名
func AvgBy[T comparable, R Integer | Float](q Query[T], selector func(T) R) float64 {
	return AverageBy(q, selector)
}

// Contains 判断序列中是否包含指定的元素
func Contains[T comparable](q Query[T], value T) bool {
	return q.AnyWith(func(t T) bool { return t == value })
}

// IndexOf 返回元素的索引
func IndexOf[T comparable](q Query[T], value T) int {
	index := 0
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			if item == value {
				return index
			}
			index++
		}
	} else {
		for item := range q.iterate {
			if item == value {
				return index
			}
			index++
		}
	}
	return -1
}

// LastIndexOf 返回元素最后出现的索引
func LastIndexOf[T comparable](q Query[T], value T) int {
	index := 0
	last := -1
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			if item == value {
				last = index
			}
			index++
		}
	} else {
		for item := range q.iterate {
			if item == value {
				last = index
			}
			index++
		}
	}
	return last
}

// Distinct 过滤掉重复的元素
func Distinct[T comparable](q Query[T]) Query[T] {
	capHint := q.capacity/2 + 1
	result := Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[T]struct{}, capHint)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
				return
			}
			for item := range q.iterate {
				if _, ok := seen[item]; !ok {
					seen[item] = struct{}{}
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: q.capacity,
	}
	if q.fastSlice != nil && q.fastWhere == nil {
		return result
	}
	result.materialize = func() []T {
		items := make([]T, 0, capHint)
		seen := make(map[T]struct{}, capHint)
		if q.fastSlice != nil {
			for _, item := range q.fastSlice {
				if q.fastWhere != nil && !q.fastWhere(item) {
					continue
				}
				if _, ok := seen[item]; !ok {
					seen[item] = struct{}{}
					items = append(items, item)
				}
			}
			return items
		}
		for item := range q.iterate {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				items = append(items, item)
			}
		}
		return items
	}
	return result
}

// DistinctBy 根据键选择器过滤重复元素
func DistinctBy[T, K comparable](q Query[T], selector func(T) K) Query[T] {
	capHint := q.capacity/2 + 1
	result := Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{}, capHint)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
				return
			}
			for item := range q.iterate {
				key := selector(item)
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: q.capacity,
	}
	if q.fastSlice != nil && q.fastWhere == nil {
		return result
	}
	result.materialize = func() []T {
		items := make([]T, 0, capHint)
		seen := make(map[K]struct{}, capHint)
		if q.fastSlice != nil {
			for _, item := range q.fastSlice {
				if q.fastWhere != nil && !q.fastWhere(item) {
					continue
				}
				key := selector(item)
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					items = append(items, item)
				}
			}
			return items
		}
		for item := range q.iterate {
			key := selector(item)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				items = append(items, item)
			}
		}
		return items
	}
	return result
}

// Intersect 获取两个序列的交集
func Intersect[T comparable](q1, q2 Query[T]) Query[T] {
	capHint := q1.capacity
	if capHint <= 0 || (q2.capacity > 0 && q2.capacity < capHint) {
		capHint = q2.capacity
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[T]uint8, q2.capacity)
			if q2.fastSlice != nil {
				if q2.fastWhere == nil {
					for _, item := range q2.fastSlice {
						seen[item] = 1
					}
				} else {
					where := q2.fastWhere
					for _, item := range q2.fastSlice {
						if !where(item) {
							continue
						}
						seen[item] = 1
					}
				}
			} else {
				for item := range q2.iterate {
					seen[item] = 1
				}
			}
			if q1.fastSlice != nil {
				if q1.fastWhere == nil {
					for _, item := range q1.fastSlice {
						if seen[item] == 1 {
							seen[item] = 2
							if !yield(item) {
								return
							}
						}
					}
				} else {
					where := q1.fastWhere
					for _, item := range q1.fastSlice {
						if !where(item) {
							continue
						}
						if seen[item] == 1 {
							seen[item] = 2
							if !yield(item) {
								return
							}
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				if seen[item] == 1 {
					seen[item] = 2
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: capHint,
		materialize: func() []T {
			result := make([]T, 0, capHint)
			seen := make(map[T]uint8, q2.capacity)
			if q2.fastSlice != nil {
				if q2.fastWhere == nil {
					for _, item := range q2.fastSlice {
						seen[item] = 1
					}
				} else {
					where := q2.fastWhere
					for _, item := range q2.fastSlice {
						if !where(item) {
							continue
						}
						seen[item] = 1
					}
				}
			} else {
				for item := range q2.iterate {
					seen[item] = 1
				}
			}
			if q1.fastSlice != nil {
				if q1.fastWhere == nil {
					for _, item := range q1.fastSlice {
						if seen[item] == 1 {
							seen[item] = 2
							result = append(result, item)
						}
					}
				} else {
					where := q1.fastWhere
					for _, item := range q1.fastSlice {
						if !where(item) {
							continue
						}
						if seen[item] == 1 {
							seen[item] = 2
							result = append(result, item)
						}
					}
				}
				return result
			}
			for item := range q1.iterate {
				if seen[item] == 1 {
					seen[item] = 2
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// IntersectBy 根据键选择器获取两个序列的交集
func IntersectBy[T, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	capHint := q1.capacity
	if capHint <= 0 || (q2.capacity > 0 && q2.capacity < capHint) {
		capHint = q2.capacity
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[K]uint8, q2.capacity)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if seen[key] == 1 {
						seen[key] = 2
						if !yield(item) {
							return
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				key := selector(item)
				if seen[key] == 1 {
					seen[key] = 2
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: capHint,
		materialize: func() []T {
			result := make([]T, 0, capHint)
			seen := make(map[K]uint8, q2.capacity)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if seen[key] == 1 {
						seen[key] = 2
						result = append(result, item)
					}
				}
				return result
			}
			for item := range q1.iterate {
				key := selector(item)
				if seen[key] == 1 {
					seen[key] = 2
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// Union 获取两个序列的并集
func Union[T comparable](q1, q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[T]struct{}, q1.capacity+q2.capacity)
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			} else {
				for item := range q1.iterate {
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
				return
			}
			for item := range q2.iterate {
				if _, ok := seen[item]; !ok {
					seen[item] = struct{}{}
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: q1.capacity + q2.capacity,
		materialize: func() []T {
			result := make([]T, 0, q1.capacity+q2.capacity)
			seen := make(map[T]struct{}, q1.capacity+q2.capacity)
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						result = append(result, item)
					}
				}
			} else {
				for item := range q1.iterate {
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						result = append(result, item)
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					if _, ok := seen[item]; !ok {
						seen[item] = struct{}{}
						result = append(result, item)
					}
				}
				return result
			}
			for item := range q2.iterate {
				if _, ok := seen[item]; !ok {
					seen[item] = struct{}{}
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// UnionBy 根据键选择器获取两个序列的并集
func UnionBy[T, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{}, q1.capacity+q2.capacity)
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			} else {
				for item := range q1.iterate {
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			} else {
				for item := range q2.iterate {
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			}
		},
		capacity: q1.capacity + q2.capacity,
		materialize: func() []T {
			result := make([]T, 0, q1.capacity+q2.capacity)
			seen := make(map[K]struct{}, q1.capacity+q2.capacity)
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						result = append(result, item)
					}
				}
			} else {
				for item := range q1.iterate {
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						result = append(result, item)
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						result = append(result, item)
					}
				}
			} else {
				for item := range q2.iterate {
					key := selector(item)
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						result = append(result, item)
					}
				}
			}
			return result
		},
	}
}

// Except 获取两个序列的差集 (q1 中有而 q2 中没有)
func Except[T comparable](q1, q2 Query[T]) Query[T] {
	capHint := q2.capacity + q1.capacity/2 + 1
	return Query[T]{
		iterate: func(yield func(T) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已从 q1 输出
			seen := make(map[T]uint8, capHint)
			if q2.fastSlice != nil {
				if q2.fastWhere == nil {
					for _, item := range q2.fastSlice {
						seen[item] = 1
					}
				} else {
					where := q2.fastWhere
					for _, item := range q2.fastSlice {
						if !where(item) {
							continue
						}
						seen[item] = 1
					}
				}
			} else {
				for item := range q2.iterate {
					seen[item] = 1
				}
			}
			if q1.fastSlice != nil {
				if q1.fastWhere == nil {
					for _, item := range q1.fastSlice {
						if seen[item] == 0 {
							seen[item] = 2
							if !yield(item) {
								return
							}
						}
					}
				} else {
					where := q1.fastWhere
					for _, item := range q1.fastSlice {
						if !where(item) {
							continue
						}
						if seen[item] == 0 {
							seen[item] = 2
							if !yield(item) {
								return
							}
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				if seen[item] == 0 {
					seen[item] = 2
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: q1.capacity,
		materialize: func() []T {
			result := make([]T, 0, q1.capacity)
			seen := make(map[T]uint8, capHint)
			if q2.fastSlice != nil {
				if q2.fastWhere == nil {
					for _, item := range q2.fastSlice {
						seen[item] = 1
					}
				} else {
					where := q2.fastWhere
					for _, item := range q2.fastSlice {
						if !where(item) {
							continue
						}
						seen[item] = 1
					}
				}
			} else {
				for item := range q2.iterate {
					seen[item] = 1
				}
			}
			if q1.fastSlice != nil {
				if q1.fastWhere == nil {
					for _, item := range q1.fastSlice {
						if seen[item] == 0 {
							seen[item] = 2
							result = append(result, item)
						}
					}
				} else {
					where := q1.fastWhere
					for _, item := range q1.fastSlice {
						if !where(item) {
							continue
						}
						if seen[item] == 0 {
							seen[item] = 2
							result = append(result, item)
						}
					}
				}
				return result
			}
			for item := range q1.iterate {
				if seen[item] == 0 {
					seen[item] = 2
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// ExceptBy 根据键选择器获取两个序列的差集
func ExceptBy[T, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	capHint := q2.capacity + q1.capacity/2 + 1
	return Query[T]{
		iterate: func(yield func(T) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已从 q1 输出
			seen := make(map[K]uint8, capHint)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if seen[key] == 0 {
						seen[key] = 2
						if !yield(item) {
							return
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				key := selector(item)
				if seen[key] == 0 {
					seen[key] = 2
					if !yield(item) {
						return
					}
				}
			}
		},
		capacity: q1.capacity,
		materialize: func() []T {
			result := make([]T, 0, q1.capacity)
			seen := make(map[K]uint8, capHint)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if seen[key] == 0 {
						seen[key] = 2
						result = append(result, item)
					}
				}
				return result
			}
			for item := range q1.iterate {
				key := selector(item)
				if seen[key] == 0 {
					seen[key] = 2
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// Select 将序列中的每个元素投影到新表单
func Select[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	result := Query[V]{
		iterate: func(yield func(V) bool) {
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					if !yield(selector(item)) {
						return
					}
				}
				return
			}
			for item := range q.iterate {
				if !yield(selector(item)) {
					return
				}
			}
		},
		capacity: q.capacity,
	}
	result.materialize = func() []V {
		capHint := q.capacity
		if q.fastWhere != nil {
			capHint = q.capacity/2 + 1
		}
		if capHint < 0 {
			capHint = 0
		}
		if q.fastSlice != nil && q.fastWhere == nil {
			return SliceMap(q.fastSlice, selector)
		}
		out := make([]V, 0, capHint)
		if q.fastSlice != nil {
			for _, item := range q.fastSlice {
				if q.fastWhere != nil && !q.fastWhere(item) {
					continue
				}
				out = append(out, selector(item))
			}
			return out
		}
		for item := range q.iterate {
			out = append(out, selector(item))
		}
		return out
	}
	return result
}

// SelectAsyncCtx 并发转换元素并返回一个无序序列，若包含 panic 则终止。
func SelectAsyncCtx[T, V comparable](ctx context.Context, q Query[T], selector func(T) V, workers ...int) Query[V] {
	if ctx == nil {
		ctx = context.Background()
	}
	iworkers := 1
	if len(workers) > 0 && workers[0] > 0 {
		iworkers = workers[0]
	}
	return Query[V]{
		iterate: func(yield func(V) bool) {
			workerCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			jobs := make(chan T, iworkers)
			outCh := make(chan V, iworkers)
			errCh := make(chan any, 1) // 捕获并发 worker 的 panic

			var wg sync.WaitGroup
			for i := 0; i < iworkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							select {
							case errCh <- r:
							default:
							}
							cancel()
						}
					}()
					for {
						select {
						case <-workerCtx.Done():
							return
						case item, ok := <-jobs:
							if !ok {
								return
							}
							val := selector(item)
							select {
							case <-workerCtx.Done():
								return
							case outCh <- val:
							}
						}
					}
				}()
			}

			// 生产者
			go func() {
				defer close(jobs)
				if q.fastSlice != nil {
					for _, item := range q.fastSlice {
						if q.fastWhere != nil && !q.fastWhere(item) {
							continue
						}
						select {
						case <-workerCtx.Done():
							return
						case jobs <- item:
						}
					}
					return
				}
				for item := range q.iterate {
					select {
					case <-workerCtx.Done():
						return
					case jobs <- item:
					}
				}
			}()

			// 关闭输出
			go func() {
				wg.Wait()
				close(outCh)
			}()

			panicIfAny := func() {
				select {
				case panicErr := <-errCh:
					panic(panicErr)
				default:
				}
			}

			for {
				select {
				case <-workerCtx.Done():
					panicIfAny()
					return
				case val, ok := <-outCh:
					if !ok {
						panicIfAny()
						return
					}
					if !yield(val) {
						cancel()
						return
					}
				}
			}
		},
	}
}

// GroupBy 根据键选择器将元素分组
func GroupBy[T, K comparable](q Query[T], keySelector func(T) K) Query[*KV[K, []T]] {
	return Query[*KV[K, []T]]{
		iterate: func(yield func(*KV[K, []T]) bool) {
			groups := make(map[K][]T, q.capacity)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					key := keySelector(item)
					groups[key] = append(groups[key], item)
				}
			} else {
				for item := range q.iterate {
					key := keySelector(item)
					groups[key] = append(groups[key], item)
				}
			}
			for k, v := range groups {
				if !yield(&KV[K, []T]{Key: k, Value: v}) {
					return
				}
			}
		},
	}
}

// GroupBySelect 先分组后对每组内元素做映射
func GroupBySelect[T, K, V comparable](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[*KV[K, []V]] {
	return Query[*KV[K, []V]]{
		iterate: func(yield func(*KV[K, []V]) bool) {
			groups := make(map[K][]V, q.capacity)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					key := keySelector(item)
					groups[key] = append(groups[key], elementSelector(item))
				}
			} else {
				for item := range q.iterate {
					key := keySelector(item)
					groups[key] = append(groups[key], elementSelector(item))
				}
			}
			for k, v := range groups {
				if !yield(&KV[K, []V]{Key: k, Value: v}) {
					return
				}
			}
		},
	}
}

// ToMap 根据选择器将序列转为 Map
func ToMap[T, K comparable](q Query[T], keySelector func(T) K) map[K]T {
	m := make(map[K]T, q.capacity)
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			m[keySelector(item)] = item
		}
		return m
	}
	for item := range q.iterate {
		m[keySelector(item)] = item
	}
	return m
}

// ToMapSelect 根据键选择器和值选择器转换序列
func ToMapSelect[T, K, V comparable](q Query[T], keySelector func(T) K, valueSelector func(T) V) map[K]V {
	m := make(map[K]V, q.capacity)
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			m[keySelector(item)] = valueSelector(item)
		}
		return m
	}
	for item := range q.iterate {
		m[keySelector(item)] = valueSelector(item)
	}
	return m
}

// SelectAsync 并发转换元素而无需手动传递 context
func SelectAsync[T, V comparable](q Query[T], selector func(T) V, workers ...int) Query[V] {
	return SelectAsyncCtx(context.Background(), q, selector, workers...)
}

// WhereSelect 选择满足条件并执行变换的元素
func WhereSelect[T, V comparable](q Query[T], selector func(T) (V, bool)) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			if q.fastSlice != nil {
				if q.fastWhere == nil {
					for _, item := range q.fastSlice {
						val, ok := selector(item)
						if ok {
							if !yield(val) {
								return
							}
						}
					}
				} else {
					where := q.fastWhere
					for _, item := range q.fastSlice {
						if !where(item) {
							continue
						}
						val, ok := selector(item)
						if ok {
							if !yield(val) {
								return
							}
						}
					}
				}
				return
			}
			for item := range q.iterate {
				val, ok := selector(item)
				if ok {
					if !yield(val) {
						return
					}
				}
			}
		},
		capacity: q.capacity,
		materialize: func() []V {
			result := make([]V, 0, q.capacity/2+1)
			if q.fastSlice != nil {
				if q.fastWhere == nil {
					for _, item := range q.fastSlice {
						val, ok := selector(item)
						if ok {
							result = append(result, val)
						}
					}
					return result
				}
				where := q.fastWhere
				for _, item := range q.fastSlice {
					if !where(item) {
						continue
					}
					val, ok := selector(item)
					if ok {
						result = append(result, val)
					}
				}
				return result
			}
			for item := range q.iterate {
				val, ok := selector(item)
				if ok {
					result = append(result, val)
				}
			}
			return result
		},
	}
}

// DistinctSelect 映射并去重
func DistinctSelect[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	capHint := q.capacity/2 + 1
	result := Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{}, capHint)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
				return
			}
			for item := range q.iterate {
				val := selector(item)
				if _, ok := seen[val]; !ok {
					seen[val] = struct{}{}
					if !yield(val) {
						return
					}
				}
			}
		},
		capacity: q.capacity,
	}
	if q.fastSlice != nil && q.fastWhere == nil {
		return result
	}
	result.materialize = func() []V {
		items := make([]V, 0, capHint)
		seen := make(map[V]struct{}, capHint)
		if q.fastSlice != nil {
			for _, item := range q.fastSlice {
				if q.fastWhere != nil && !q.fastWhere(item) {
					continue
				}
				val := selector(item)
				if _, ok := seen[val]; !ok {
					seen[val] = struct{}{}
					items = append(items, val)
				}
			}
			return items
		}
		for item := range q.iterate {
			val := selector(item)
			if _, ok := seen[val]; !ok {
				seen[val] = struct{}{}
				items = append(items, val)
			}
		}
		return items
	}
	return result
}

// UnionSelect 映射并合并去重
func UnionSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{}, q.capacity+q2.capacity)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			} else {
				for item := range q.iterate {
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			} else {
				for item := range q2.iterate {
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			}
		},
		capacity: q.capacity + q2.capacity,
		materialize: func() []V {
			result := make([]V, 0, q.capacity+q2.capacity)
			seen := make(map[V]struct{}, q.capacity+q2.capacity)
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						result = append(result, val)
					}
				}
			} else {
				for item := range q.iterate {
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						result = append(result, val)
					}
				}
			}
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						result = append(result, val)
					}
				}
			} else {
				for item := range q2.iterate {
					val := selector(item)
					if _, ok := seen[val]; !ok {
						seen[val] = struct{}{}
						result = append(result, val)
					}
				}
			}
			return result
		},
	}
}

// IntersectSelect 映射并取交集去重
func IntersectSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	capHint := q.capacity
	if capHint <= 0 || (q2.capacity > 0 && q2.capacity < capHint) {
		capHint = q2.capacity
	}
	return Query[V]{
		iterate: func(yield func(V) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[V]uint8, q2.capacity)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if seen[val] == 1 {
						seen[val] = 2
						if !yield(val) {
							return
						}
					}
				}
				return
			}
			for item := range q.iterate {
				val := selector(item)
				if seen[val] == 1 {
					seen[val] = 2
					if !yield(val) {
						return
					}
				}
			}
		},
		capacity: capHint,
		materialize: func() []V {
			result := make([]V, 0, capHint)
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[V]uint8, q2.capacity)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if seen[val] == 1 {
						seen[val] = 2
						result = append(result, val)
					}
				}
				return result
			}
			for item := range q.iterate {
				val := selector(item)
				if seen[val] == 1 {
					seen[val] = 2
					result = append(result, val)
				}
			}
			return result
		},
	}
}

// ExceptSelect 映射并取差集去重
func ExceptSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	capHint := q2.capacity + q.capacity/2 + 1
	return Query[V]{
		iterate: func(yield func(V) bool) {
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[V]uint8, capHint)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if seen[val] == 0 {
						seen[val] = 2
						if !yield(val) {
							return
						}
					}
				}
				return
			}
			for item := range q.iterate {
				val := selector(item)
				if seen[val] == 0 {
					seen[val] = 2
					if !yield(val) {
						return
					}
				}
			}
		},
		capacity: q.capacity,
		materialize: func() []V {
			result := make([]V, 0, q.capacity)
			// 0: 不存在 1: 存在于 q2 2: 已输出
			seen := make(map[V]uint8, capHint)
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = 1
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = 1
				}
			}
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					val := selector(item)
					if seen[val] == 0 {
						seen[val] = 2
						result = append(result, val)
					}
				}
				return result
			}
			for item := range q.iterate {
				val := selector(item)
				if seen[val] == 0 {
					seen[val] = 2
					result = append(result, val)
				}
			}
			return result
		},
	}
}
