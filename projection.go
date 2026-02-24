package linq

import (
	"context"
	"sync"
)

// Select 将序列中的每个元素投影到新表单
func Select[T, V any](q Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			for item := range q.iterate {
				if !yield(selector(item)) {
					return
				}
			}
		},
	}
}

// SelectAsyncCtx 并发转换元素并返回一个无序序列，若包含 panic 则终止。
func SelectAsyncCtx[T, V any](ctx context.Context, q Query[T], workers int, selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			outCh := make(chan V)
			errCh := make(chan any, workers) // 捕获并发协程的 panic

			var wg sync.WaitGroup
			workerCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			// 生产者协程
			go func() {
				defer close(outCh)
				sem := make(chan struct{}, workers)
				for item := range q.iterate {
					select {
					case <-workerCtx.Done():
						return
					case sem <- struct{}{}:
					}

					wg.Add(1)
					go func(val T) {
						defer wg.Done()
						defer func() { <-sem }()
						defer func() {
							if r := recover(); r != nil {
								select {
								case errCh <- r:
									cancel()
								default:
								}
							}
						}()
						res := selector(val)
						select {
						case <-workerCtx.Done():
							return
						case outCh <- res:
						}
					}(item)
				}
				wg.Wait()
			}()

			// 迭代器消费者
			for {
				select {
				case <-ctx.Done():
					return
				case panicErr := <-errCh:
					panic(panicErr) // 向外抛出捕获到的 panic
				case val, ok := <-outCh:
					if !ok {
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
func GroupBy[T any, K comparable](q Query[T], keySelector func(T) K) Query[KV[K, []T]] {
	return Query[KV[K, []T]]{
		iterate: func(yield func(KV[K, []T]) bool) {
			groups := make(map[K][]T)
			for item := range q.iterate {
				key := keySelector(item)
				groups[key] = append(groups[key], item)
			}
			for k, v := range groups {
				if !yield(KV[K, []T]{Key: k, Value: v}) {
					return
				}
			}
		},
	}
}

// GroupBySelect 先分组后对每组内元素做映射
func GroupBySelect[T any, K comparable, V any](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[KV[K, []V]] {
	return Query[KV[K, []V]]{
		iterate: func(yield func(KV[K, []V]) bool) {
			groups := make(map[K][]V)
			for item := range q.iterate {
				key := keySelector(item)
				groups[key] = append(groups[key], elementSelector(item))
			}
			for k, v := range groups {
				if !yield(KV[K, []V]{Key: k, Value: v}) {
					return
				}
			}
		},
	}
}

// ToMap 根据选择器将序列转为 Map
func ToMap[T any, K comparable](q Query[T], keySelector func(T) K) map[K]T {
	m := make(map[K]T, q.capacity)
	for item := range q.iterate {
		m[keySelector(item)] = item
	}
	return m
}

// ToMapSelect 根据键选择器和值选择器转换序列
func ToMapSelect[T any, K comparable, V any](q Query[T], keySelector func(T) K, valueSelector func(T) V) map[K]V {
	m := make(map[K]V, q.capacity)
	for item := range q.iterate {
		m[keySelector(item)] = valueSelector(item)
	}
	return m
}

// Try 执行可能会引发 panic 的函数
func Try[T any](f func() T) (result T, err any) {
	defer func() {
		if r := recover(); r != nil {
			err = r
		}
	}()
	result = f()
	return
}

// SelectAsync 并发转换元素而无需手动传递 context
func SelectAsync[T, V comparable](q Query[T], workers int, selector func(T) V) Query[V] {
	return SelectAsyncCtx(context.Background(), q, workers, selector)
}

// WhereSelect 选择满足条件并执行变换的元素
func WhereSelect[T, V comparable](q Query[T], selector func(T) (V, bool)) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			for item := range q.iterate {
				val, ok := selector(item)
				if ok {
					if !yield(val) {
						return
					}
				}
			}
		},
	}
}

// DistinctSelect 映射并去重
func DistinctSelect[T any, V comparable](q Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{})
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
	}
}

// UnionSelect 映射并合并去重
func UnionSelect[T any, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{})
			for item := range q.iterate {
				val := selector(item)
				if _, ok := seen[val]; !ok {
					seen[val] = struct{}{}
					if !yield(val) {
						return
					}
				}
			}
			for item := range q2.iterate {
				val := selector(item)
				if _, ok := seen[val]; !ok {
					seen[val] = struct{}{}
					if !yield(val) {
						return
					}
				}
			}
		},
	}
}

// IntersectSelect 映射并取交集去重
func IntersectSelect[T any, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{})
			for item := range q2.iterate {
				seen[selector(item)] = struct{}{}
			}
			emitted := make(map[V]struct{})
			for item := range q.iterate {
				val := selector(item)
				if _, ok := seen[val]; ok {
					if _, already := emitted[val]; !already {
						emitted[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			}
		},
	}
}

// ExceptSelect 映射并取差集去重
func ExceptSelect[T any, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func(yield func(V) bool) {
			seen := make(map[V]struct{})
			for item := range q2.iterate {
				seen[selector(item)] = struct{}{}
			}
			emitted := make(map[V]struct{})
			for item := range q.iterate {
				val := selector(item)
				if _, ok := seen[val]; !ok {
					if _, already := emitted[val]; !already {
						emitted[val] = struct{}{}
						if !yield(val) {
							return
						}
					}
				}
			}
		},
	}
}
