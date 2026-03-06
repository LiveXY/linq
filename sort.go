package linq

import (
	"cmp"
	"slices"
)

// HasOrder 判断查询目前是否已定义排序规则
func (q Query[T]) HasOrder() bool {
	return q.compare != nil || len(q.sortCompares) > 0
}

// OrderBy 指定主要排序键，按升序对序列元素进行排序
func OrderBy[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderBy(q, func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	})
}

// OrderByDescending 指定主要排序键，按降序对序列元素进行排序
func OrderByDescending[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderBy(q, func(a, b T) int {
		return cmp.Compare(key(b), key(a)) // 降序关键：b 与 a 比较
	})
}

// OrderByUnstable 指定主要排序键，按升序进行不稳定排序
func OrderByUnstable[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderByUnstable(q, func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	})
}

// OrderByDescendingUnstable 指定主要排序键，按降序进行不稳定排序
func OrderByDescendingUnstable[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderByUnstable(q, func(a, b T) int {
		return cmp.Compare(key(b), key(a))
	})
}

// ThenBy 指定次要排序键，按升序对序列元素进行后续排序
func ThenBy[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	if !q.HasOrder() {
		return q
	}
	nextCmp := func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	}
	return orderBy(q, nextCmp)
}

// ThenByDescending 指定次要排序键，按降序对序列元素进行后续排序
func ThenByDescending[T comparable, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	if !q.HasOrder() {
		return q
	}
	nextCmp := func(a, b T) int {
		return cmp.Compare(key(b), key(a))
	}
	return orderBy(q, nextCmp)
}

// 组合比较器：按优先级依次比较
func composeComparators[T comparable](comparators []CompareFunc[T]) CompareFunc[T] {
	switch len(comparators) {
	case 0:
		return nil
	case 1:
		return comparators[0]
	case 2:
		c0, c1 := comparators[0], comparators[1]
		return func(a, b T) int {
			if r := c0(a, b); r != 0 {
				return r
			}
			return c1(a, b)
		}
	case 3:
		c0, c1, c2 := comparators[0], comparators[1], comparators[2]
		return func(a, b T) int {
			if r := c0(a, b); r != 0 {
				return r
			}
			if r := c1(a, b); r != 0 {
				return r
			}
			return c2(a, b)
		}
	default:
		return func(a, b T) int {
			for _, cmpFn := range comparators {
				if r := cmpFn(a, b); r != 0 {
					return r
				}
			}
			return 0
		}
	}
}

// 核心排序执行函数
func orderBy[T comparable](q Query[T], cmpFn CompareFunc[T]) Query[T] {
	return orderByWithMode(q, cmpFn, true)
}

// 核心不稳定排序执行函数
func orderByUnstable[T comparable](q Query[T], cmpFn CompareFunc[T]) Query[T] {
	return orderByWithMode(q, cmpFn, false)
}

func orderByWithMode[T comparable](q Query[T], cmpFn CompareFunc[T], stable bool) Query[T] {
	var source Query[T]
	if q.sortSource != nil {
		source = *q.sortSource
	} else {
		source = q
	}

	comparators := make([]CompareFunc[T], 0, len(q.sortCompares)+1)
	if len(q.sortCompares) > 0 {
		comparators = append(comparators, q.sortCompares...)
	} else if q.compare != nil {
		comparators = append(comparators, q.compare)
	}
	comparators = append(comparators, cmpFn)
	combinedCmp := composeComparators(comparators)

	sortStable := stable
	if q.sortSource != nil {
		sortStable = q.sortStable
	}

	materialize := func() []T {
		data := source.ToSlice()
		if combinedCmp == nil || len(data) <= 1 {
			return data
		}
		if sortStable {
			slices.SortStableFunc(data, combinedCmp)
		} else {
			slices.SortFunc(data, combinedCmp)
		}
		return data
	}
	return Query[T]{
		compare: combinedCmp,
		iterate: func(yield func(T) bool) {
			data := materialize()
			for _, item := range data {
				if !yield(item) {
					return
				}
			}
		},
		capacity:     source.capacity,
		materialize:  materialize,
		sortSource:   &source,
		sortCompares: comparators,
		sortStable:   sortStable,
	}
}

// OrderedQuery 包含已有的排序规则，供特定场景复用
type OrderedQuery[T comparable] struct {
	Query[T]
	sortCompares []CompareFunc[T]
	sortStable   bool
}

// Order 指定排序规则
func (q Query[T]) Order(comparator CompareFunc[T]) OrderedQuery[T] {
	return OrderedQuery[T]{
		Query:        q,
		sortCompares: []CompareFunc[T]{comparator},
		sortStable:   true,
	}
}

// OrderUnstable 指定排序规则并使用不稳定排序
func (q Query[T]) OrderUnstable(comparator CompareFunc[T]) OrderedQuery[T] {
	return OrderedQuery[T]{
		Query:        q,
		sortCompares: []CompareFunc[T]{comparator},
		sortStable:   false,
	}
}

// Asc 根据键选择器生成升序比较器
func Asc[T comparable, K cmp.Ordered](selector func(T) K) CompareFunc[T] {
	return func(a, b T) int {
		return cmp.Compare(selector(a), selector(b))
	}
}

// Desc 根据键选择器生成降序比较器
func Desc[T comparable, K cmp.Ordered](selector func(T) K) CompareFunc[T] {
	return func(a, b T) int {
		return cmp.Compare(selector(b), selector(a))
	}
}

// Then 添加后续排序规则
func (oq OrderedQuery[T]) Then(comparator CompareFunc[T]) OrderedQuery[T] {
	comparators := make([]CompareFunc[T], 0, len(oq.sortCompares)+1)
	if len(oq.sortCompares) > 0 {
		comparators = append(comparators, oq.sortCompares...)
	} else if oq.Query.compare != nil {
		comparators = append(comparators, oq.Query.compare)
	}
	comparators = append(comparators, comparator)

	stable := oq.sortStable
	if len(oq.sortCompares) == 0 && oq.Query.compare == nil {
		stable = true
	}

	return OrderedQuery[T]{
		Query:        oq.Query,
		sortCompares: comparators,
		sortStable:   stable,
	}
}

// ToQuery 将 OrderedQuery 转换为已排序的 Query
func (oq OrderedQuery[T]) ToQuery() Query[T] {
	return From(oq.sortedSlice())
}

// ToSlice 提供已排序结果
func (oq OrderedQuery[T]) ToSlice() []T {
	return oq.sortedSlice()
}

func (oq OrderedQuery[T]) sortedSlice() []T {
	data := oq.Query.ToSlice()
	cmpFn := composeComparators(oq.sortCompares)
	if cmpFn == nil || len(data) <= 1 {
		return data
	}
	if oq.sortStable {
		slices.SortStableFunc(data, cmpFn)
	} else {
		slices.SortFunc(data, cmpFn)
	}
	return data
}

// First 返回已排序第一个元素
func (oq OrderedQuery[T]) First() T {
	return oq.ToQuery().First()
}

// Last 返回已排序最后一个元素
func (oq OrderedQuery[T]) Last() T {
	return oq.ToQuery().Last()
}

// Take 代理
func (oq OrderedQuery[T]) Take(count int) Query[T] {
	return oq.ToQuery().Take(count)
}

// Skip 代理
func (oq OrderedQuery[T]) Skip(count int) Query[T] {
	return oq.ToQuery().Skip(count)
}

// Where 代理
func (oq OrderedQuery[T]) Where(predicate func(T) bool) Query[T] {
	return oq.ToQuery().Where(predicate)
}

// TakeWhile 代理
func (oq OrderedQuery[T]) TakeWhile(predicate func(T) bool) Query[T] {
	return oq.ToQuery().TakeWhile(predicate)
}

// SkipWhile 代理
func (oq OrderedQuery[T]) SkipWhile(predicate func(T) bool) Query[T] {
	return oq.ToQuery().SkipWhile(predicate)
}

// IndexOfWith 代理
func (oq OrderedQuery[T]) IndexOfWith(predicate func(T) bool) int {
	return oq.ToQuery().IndexOfWith(predicate)
}

// ForEach 代理
func (oq OrderedQuery[T]) ForEach(action func(T) bool) {
	oq.ToQuery().ForEach(action)
}

// Reverse 代理
func (oq OrderedQuery[T]) Reverse() Query[T] {
	return oq.ToQuery().Reverse()
}

// Append 代理
func (oq OrderedQuery[T]) Append(item T) Query[T] {
	return oq.ToQuery().Append(item)
}

// Prepend 代理
func (oq OrderedQuery[T]) Prepend(item T) Query[T] {
	return oq.ToQuery().Prepend(item)
}

// DefaultIfEmpty 代理
func (oq OrderedQuery[T]) DefaultIfEmpty(defaultValue T) Query[T] {
	return oq.ToQuery().DefaultIfEmpty(defaultValue)
}

// Page 代理
func (oq OrderedQuery[T]) Page(pageNumber, pageSize int) Query[T] {
	return oq.ToQuery().Page(pageNumber, pageSize)
}

// FirstDefault 代理
func (oq OrderedQuery[T]) FirstDefault(defaultValue ...T) T {
	return oq.ToQuery().FirstDefault(defaultValue...)
}

// LastDefault 代理
func (oq OrderedQuery[T]) LastDefault(defaultValue ...T) T {
	return oq.ToQuery().LastDefault(defaultValue...)
}

// ForEachIndexed 代理
func (oq OrderedQuery[T]) ForEachIndexed(action func(int, T) bool) {
	oq.ToQuery().ForEachIndexed(action)
}

// Distinct 代理 (仅当 T 可比较时有效)
func (oq OrderedQuery[T]) Distinct() Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[T]struct{})
			for item := range oq.ToQuery().iterate {
				if _, ok := seen[item]; !ok {
					seen[item] = struct{}{}
					if !yield(item) {
						return
					}
				}
			}
		},
	}
}

// IndexOf 代理 (仅当 T 可比较时有效)
func (oq OrderedQuery[T]) IndexOf(value T) int {
	index := 0
	for item := range oq.ToQuery().iterate {
		if item == value {
			return index
		}
		index++
	}
	return -1
}
