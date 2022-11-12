package linq

import (
	"sort"

	"golang.org/x/exp/constraints"
)

func (q Query[T]) HasOrder() bool {
	return q.lesser != nil
}

func OrderBy[T any, K constraints.Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	})
}
func OrderByDescending[T any, K constraints.Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) > key(data[j])
		}
	})
}
func ThenBy[T any, K constraints.Ordered](q Query[T], key func(t T) K) Query[T] {
	lesser := q.lesser
	return orderByLesser(q, chainLessers(lesser, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	}))
}
func ThenByDescending[T any, K constraints.Ordered](q Query[T], key func(t T) K) Query[T] {
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
