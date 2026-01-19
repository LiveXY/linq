package linq

import (
	"sort"
)

// HasOrder 判断查询目前是否已定义排序规则
func (q Query[T]) HasOrder() bool {
	return q.lesser != nil
}

// OrderBy 指定主要排序键，按升序对序列元素进行排序
func OrderBy[T any, K Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	})
}

// OrderByDescending 指定主要排序键，按降序对序列元素进行排序
func OrderByDescending[T any, K Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) > key(data[j])
		}
	})
}

// ThenBy 指定次要排序键，按升序对序列元素进行后续排序
// 必须在 OrderBy 或 OrderByDescending 之后调用
func ThenBy[T any, K Ordered](q Query[T], key func(t T) K) Query[T] {
	lesser := q.lesser
	return orderByLesser(q, chainLessers(lesser, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	}))
}

// ThenByDescending 指定次要排序键，按降序对序列元素进行后续排序
// 必须在 OrderBy 或 OrderByDescending 之后调用
func ThenByDescending[T any, K Ordered](q Query[T], key func(t T) K) Query[T] {
	lesser := q.lesser
	return orderByLesser(q, chainLessers(lesser, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) > key(data[j])
		}
	}))
}

func chainLessers[T any](a, b lesserFunc[T]) lesserFunc[T] {
	return func(data []T) func(i, j int) bool {
		a, b := a(data), b(data)
		return func(i, j int) bool {
			return a(i, j) || !a(j, i) && b(i, j)
		}
	}
}
func orderByLesser[T any](q Query[T], lesser lesserFunc[T]) Query[T] {
	return Query[T]{
		lesser: lesser,
		iterate: func() func() (T, bool) {
			data := q.ToSlice()
			sort.Slice(data, lesser(data))
			return From(data).iterate()
		},
	}
}
