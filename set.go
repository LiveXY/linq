package linq

// Distinct 过滤掉重复的元素
func Distinct[T comparable](q Query[T]) Query[T] {
	return DistinctBy(q, func(t T) T { return t })
}

// DistinctBy 根据键选择器过滤重复元素
func DistinctBy[T any, K comparable](q Query[T], selector func(T) K) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{})
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
	}
}

// Intersect 获取两个序列的交集
func Intersect[T comparable](q1, q2 Query[T]) Query[T] {
	return IntersectBy(q1, q2, func(t T) T { return t })
}

// IntersectBy 根据键选择器获取两个序列的交集
func IntersectBy[T any, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{})
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = struct{}{}
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = struct{}{}
				}
			}
			emitted := make(map[K]struct{})
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; ok {
						if _, already := emitted[key]; !already {
							emitted[key] = struct{}{}
							if !yield(item) {
								return
							}
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				key := selector(item)
				if _, ok := seen[key]; ok {
					if _, already := emitted[key]; !already {
						emitted[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			}
		},
	}
}

// Union 获取两个序列的并集
func Union[T comparable](q1, q2 Query[T]) Query[T] {
	return UnionBy(q1, q2, func(t T) T { return t })
}

// UnionBy 根据键选择器获取两个序列的并集
func UnionBy[T any, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{})
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
	}
}

// Except 获取两个序列的差集 (q1 中有而 q2 中没有)
func Except[T comparable](q1, q2 Query[T]) Query[T] {
	return ExceptBy(q1, q2, func(t T) T { return t })
}

// ExceptBy 根据键选择器获取两个序列的差集
func ExceptBy[T any, K comparable](q1, q2 Query[T], selector func(T) K) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			seen := make(map[K]struct{})
			if q2.fastSlice != nil {
				for _, item := range q2.fastSlice {
					if q2.fastWhere != nil && !q2.fastWhere(item) {
						continue
					}
					seen[selector(item)] = struct{}{}
				}
			} else {
				for item := range q2.iterate {
					seen[selector(item)] = struct{}{}
				}
			}
			emitted := make(map[K]struct{})
			if q1.fastSlice != nil {
				for _, item := range q1.fastSlice {
					if q1.fastWhere != nil && !q1.fastWhere(item) {
						continue
					}
					key := selector(item)
					if _, ok := seen[key]; !ok {
						if _, already := emitted[key]; !already {
							emitted[key] = struct{}{}
							if !yield(item) {
								return
							}
						}
					}
				}
				return
			}
			for item := range q1.iterate {
				key := selector(item)
				if _, ok := seen[key]; !ok {
					if _, already := emitted[key]; !already {
						emitted[key] = struct{}{}
						if !yield(item) {
							return
						}
					}
				}
			}
		},
	}
}
