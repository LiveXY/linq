package linq

// Where 过滤元素
func (q Query[T]) Where(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		var combinedPred func(T) bool
		if q.fastWhere == nil {
			combinedPred = predicate
		} else {
			oldPred := q.fastWhere
			combinedPred = func(t T) bool { return oldPred(t) && predicate(t) }
		}
		return Query[T]{
			iterate: func(yield func(T) bool) {
				for _, item := range source {
					if combinedPred(item) {
						if !yield(item) {
							break
						}
					}
				}
			},
			fastSlice: source,
			fastWhere: combinedPred,
			capacity:  q.capacity,
		}
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for item := range q.iterate {
				if predicate(item) {
					if !yield(item) {
						break
					}
				}
			}
		},
		capacity: q.capacity,
	}
}

// Skip 跳过前 N 个元素
func (q Query[T]) Skip(count int) Query[T] {
	if q.fastSlice != nil && q.fastWhere == nil {
		if count >= len(q.fastSlice) {
			return Empty[T]()
		}
		if count <= 0 {
			return q
		}
		return From(q.fastSlice[count:])
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			n := count
			for item := range q.iterate {
				if n > 0 {
					n--
					continue
				}
				if !yield(item) {
					break
				}
			}
		},
	}
}

// Take 获取前 N 个元素
func (q Query[T]) Take(count int) Query[T] {
	if q.fastSlice != nil && q.fastWhere == nil {
		if count <= 0 {
			return Empty[T]()
		}
		if count >= len(q.fastSlice) {
			return q
		}
		return From(q.fastSlice[:count])
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			n := count
			for item := range q.iterate {
				if n <= 0 {
					break
				}
				n--
				if !yield(item) {
					break
				}
			}
		},
	}
}

// TakeWhile 获取满足条件的元素，一旦不满足则停止
func (q Query[T]) TakeWhile(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		return Query[T]{
			iterate: func(yield func(T) bool) {
				for _, item := range source {
					if preFilter != nil && !preFilter(item) {
						continue
					}
					if !predicate(item) {
						break
					}
					if !yield(item) {
						break
					}
				}
			},
		}
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for item := range q.iterate {
				if !predicate(item) {
					break
				}
				if !yield(item) {
					break
				}
			}
		},
	}
}

// SkipWhile 跳过满足条件的元素，之后全部获取
func (q Query[T]) SkipWhile(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		return Query[T]{
			iterate: func(yield func(T) bool) {
				skipping := true
				for _, item := range source {
					if preFilter != nil && !preFilter(item) {
						continue
					}
					if skipping {
						if predicate(item) {
							continue
						}
						skipping = false
					}
					if !yield(item) {
						break
					}
				}
			},
		}
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			skipping := true
			for item := range q.iterate {
				if skipping {
					if predicate(item) {
						continue
					}
					skipping = false
				}
				if !yield(item) {
					break
				}
			}
		},
	}
}

// Page 分页查询
func (q Query[T]) Page(page, pageSize int) Query[T] {
	return q.Skip((page - 1) * pageSize).Take(pageSize)
}

// Append 在序列末尾追加
func (q Query[T]) Append(item T) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			if q.fastSlice != nil {
				source := q.fastSlice
				predicate := q.fastWhere
				for _, t := range source {
					if predicate != nil && !predicate(t) {
						continue
					}
					if !yield(t) {
						return
					}
				}
			} else {
				for t := range q.iterate {
					if !yield(t) {
						return
					}
				}
			}
			yield(item)
		},
	}
}

// Prepend 在序列开头追加
func (q Query[T]) Prepend(item T) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			if !yield(item) {
				return
			}
			if q.fastSlice != nil {
				source := q.fastSlice
				predicate := q.fastWhere
				for _, t := range source {
					if predicate != nil && !predicate(t) {
						continue
					}
					if !yield(t) {
						return
					}
				}
			} else {
				for t := range q.iterate {
					if !yield(t) {
						return
					}
				}
			}
		},
	}
}

// Concat 连接两个序列
func (q Query[T]) Concat(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			if q.fastSlice != nil {
				source := q.fastSlice
				predicate := q.fastWhere
				for _, t := range source {
					if predicate != nil && !predicate(t) {
						continue
					}
					if !yield(t) {
						return
					}
				}
			} else {
				for t := range q.iterate {
					if !yield(t) {
						return
					}
				}
			}

			if q2.fastSlice != nil {
				source := q2.fastSlice
				predicate := q2.fastWhere
				for _, t := range source {
					if predicate != nil && !predicate(t) {
						continue
					}
					if !yield(t) {
						return
					}
				}
			} else {
				for t := range q2.iterate {
					if !yield(t) {
						return
					}
				}
			}
		},
	}
}

// DefaultIfEmpty 如果空则返回默认值
func (q Query[T]) DefaultIfEmpty(defaultValue T) Query[T] {
	if q.fastSlice != nil && q.fastWhere == nil {
		if len(q.fastSlice) == 0 {
			return From([]T{defaultValue})
		}
		return q
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			empty := true
			for item := range q.iterate {
				empty = false
				if !yield(item) {
					return
				}
			}
			if empty {
				yield(defaultValue)
			}
		},
	}
}
