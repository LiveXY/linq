package linq

import (
	"cmp"
	"slices"
)

// HasOrder 判断查询目前是否已定义排序规则
func (q Query[T]) HasOrder() bool {
	return q.compare != nil
}

// OrderBy 指定主要排序键，按升序对序列元素进行排序
func OrderBy[T any, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderBy(q, func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	})
}

// OrderByDescending 指定主要排序键，按降序对序列元素进行排序
func OrderByDescending[T any, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	return orderBy(q, func(a, b T) int {
		return cmp.Compare(key(b), key(a)) // 降序关键：b 与 a 比较
	})
}

// ThenBy 指定次要排序键，按升序对序列元素进行后续排序
func ThenBy[T any, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	if !q.HasOrder() {
		return q
	}
	nextCmp := func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	}
	return orderBy(q, chainComparisons(q.compare, nextCmp))
}

// ThenByDescending 指定次要排序键，按降序对序列元素进行后续排序
func ThenByDescending[T any, K cmp.Ordered](q Query[T], key func(T) K) Query[T] {
	if !q.HasOrder() {
		return q
	}
	nextCmp := func(a, b T) int {
		return cmp.Compare(key(b), key(a))
	}
	return orderBy(q, chainComparisons(q.compare, nextCmp))
}

// 链式组合比较器
func chainComparisons[T any](a, b CompareFunc[T]) CompareFunc[T] {
	return func(x, y T) int {
		if r := a(x, y); r != 0 {
			return r
		}
		return b(x, y)
	}
}

// 核心排序执行函数
func orderBy[T any](q Query[T], cmpFn CompareFunc[T]) Query[T] {
	var source Query[T]
	if q.sortSource != nil {
		source = *q.sortSource
	} else {
		source = q
	}
	return Query[T]{
		compare: cmpFn,
		iterate: func(yield func(T) bool) {
			data := source.ToSlice()
			slices.SortFunc(data, cmpFn)
			for _, item := range data {
				if !yield(item) {
					return
				}
			}
		},
		capacity:   source.capacity,
		sortSource: &source,
	}
}

// OrderedQuery 包含已有的排序规则，供特定场景复用
type OrderedQuery[T any] struct {
	Query[T]
	compare CompareFunc[T]
}

// Order 指定排序规则
func (q Query[T]) Order(comparator CompareFunc[T]) OrderedQuery[T] {
	return OrderedQuery[T]{
		Query:   q,
		compare: comparator,
	}
}

// Asc 根据键选择器生成升序比较器
func Asc[T any, K cmp.Ordered](selector func(T) K) CompareFunc[T] {
	return func(a, b T) int {
		return cmp.Compare(selector(a), selector(b))
	}
}

// Desc 根据键选择器生成降序比较器
func Desc[T any, K cmp.Ordered](selector func(T) K) CompareFunc[T] {
	return func(a, b T) int {
		return cmp.Compare(selector(b), selector(a))
	}
}

// Then 添加后续排序规则
func (oq OrderedQuery[T]) Then(comparator CompareFunc[T]) OrderedQuery[T] {
	prevCompare := oq.compare
	return OrderedQuery[T]{
		Query: oq.Query,
		compare: func(a, b T) int {
			if res := prevCompare(a, b); res != 0 {
				return res
			}
			return comparator(a, b)
		},
	}
}

// ToQuery 将 OrderedQuery 转换为已排序的 Query
func (oq OrderedQuery[T]) ToQuery() Query[T] {
	data := oq.Query.ToSlice()
	slices.SortFunc(data, oq.compare)
	return From(data)
}

// ToSlice 提供已排序结果
func (oq OrderedQuery[T]) ToSlice() []T {
	return oq.ToQuery().ToSlice()
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
			seen := make(map[any]struct{})
			for item := range oq.ToQuery().iterate {
				if _, ok := seen[any(item)]; !ok {
					seen[any(item)] = struct{}{}
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
		if any(item) == any(value) {
			return index
		}
		index++
	}
	return -1
}
