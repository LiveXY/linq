package linq

import (
	"cmp"
	"context"
	"sync"
)

// ForEach 遍历序列并对每个元素执行指定操作
func (q Query[T]) ForEach(action func(T) bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if !action(item) {
				return
			}
		}
		return
	}
	for item := range q.iterate {
		if !action(item) {
			break
		}
	}
}

// ForEachIndexed 带索引遍历序列
func (q Query[T]) ForEachIndexed(action func(int, T) bool) {
	index := 0
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if !action(index, item) {
				return
			}
			index++
		}
		return
	}
	for item := range q.iterate {
		if !action(index, item) {
			break
		}
		index++
	}
}

// ForEachParallelCtx 支持 Context 取消的并发遍历执行器（不保证顺序）
func (q Query[T]) ForEachParallelCtx(ctx context.Context, workers int, action func(T)) {
	type token struct{}
	sem := make(chan token, workers)
	var wg sync.WaitGroup

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan any, workers)

	if q.fastSlice != nil {
	loopSlice:
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			select {
			case <-workerCtx.Done():
				break loopSlice
			case sem <- token{}:
			case panicErr := <-errCh:
				if panicErr != nil {
					panic(panicErr)
				}
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

				select {
				case <-workerCtx.Done():
					return
				default:
					action(val)
				}
			}(item)
		}
	} else {
	loopIter:
		for item := range q.iterate {
			select {
			case <-workerCtx.Done():
				break loopIter
			case sem <- token{}:
			case panicErr := <-errCh:
				if panicErr != nil {
					panic(panicErr)
				}
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

				select {
				case <-workerCtx.Done():
					return
				default:
					action(val)
				}
			}(item)
		}
	}

	wg.Wait()
	close(errCh)

	// 统一抛出捕获到的 panic
	for panicErr := range errCh {
		if panicErr != nil {
			panic(panicErr)
		}
	}
}

// ForEachParallel 并发遍历无 context (底层封装 Ctx 版本)
func (q Query[T]) ForEachParallel(workers int, action func(T)) {
	q.ForEachParallelCtx(context.Background(), workers, action)
}

// MinBy 根据选择器返回最小值
func MinBy[T any, R cmp.Ordered](q Query[T], selector func(T) R) T {
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
func MaxBy[T any, R cmp.Ordered](q Query[T], selector func(T) R) T {
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
