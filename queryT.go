package linq

import (
	"context"
	"iter"
	"slices"
	"sync"
)

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
	Signed | Unsigned
}

type Float interface {
	~float32 | ~float64
}

type Complex interface {
	~complex64 | ~complex128
}

// KV 键值对结构体
type KV[K comparable, V any] struct {
	Key   K
	Value V
}

// CompareFunc 比较函数类型
type CompareFunc[T comparable] func(a, b T) int

// Query 查询结构体，是 LINQ 操作的核心类型
type Query[T comparable] struct {
	compare      CompareFunc[T]
	iterate      iter.Seq[T]
	fastSlice    []T
	fastWhere    func(T) bool
	capacity     int
	materialize  func() []T
	sortSource   *Query[T]
	sortCompares []CompareFunc[T]
	sortStable   bool
}

// Seq 返回供 for-range 从头到尾遍历的迭代器
func (q Query[T]) Seq() iter.Seq[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		if predicate == nil {
			return slices.Values(source)
		}
		return func(yield func(T) bool) {
			for _, item := range source {
				if !predicate(item) {
					continue
				}
				if !yield(item) {
					return
				}
			}
		}
	}
	return q.iterate
}

// ToSlice 将查询结果收集为切片
func (q Query[T]) ToSlice() []T {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		if predicate == nil {
			return slices.Clone(source)
		}
		result := make([]T, 0, q.capacity/2+1) // 估算
		for _, v := range source {
			if predicate(v) {
				result = append(result, v)
			}
		}
		return result
	}
	if q.materialize != nil {
		return q.materialize()
	}
	var result []T
	if q.capacity > 0 {
		result = make([]T, 0, q.capacity/2+1)
	}
	for item := range q.iterate {
		result = append(result, item)
	}
	return result
}

// ToChannel 将查询结果收集为通道，支持上下文取消
func (q Query[T]) ToChannel(ctx context.Context) <-chan T {
	if ctx == nil {
		ctx = context.Background()
	}
	ch := make(chan T)
	go func() {
		defer close(ch)
		if q.fastSlice != nil {
			for _, item := range q.fastSlice {
				if q.fastWhere != nil && !q.fastWhere(item) {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case ch <- item:
				}
			}
			return
		}
		for item := range q.iterate {
			select {
			case <-ctx.Done():
				return
			case ch <- item:
			}
		}
	}()
	return ch
}

// Reverse 返回反转后的序列的查询对象
func (q Query[T]) Reverse() Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			var items []T
			if q.fastSlice != nil {
				if q.fastWhere == nil {
					items = q.fastSlice
				} else {
					predicate := q.fastWhere
					items = make([]T, 0, q.capacity/2+1)
					for _, item := range q.fastSlice {
						if predicate(item) {
							items = append(items, item)
						}
					}
				}
			} else {
				if q.capacity > 0 {
					items = make([]T, 0, q.capacity)
				}
				for item := range q.iterate {
					items = append(items, item)
				}
			}
			for i := len(items) - 1; i >= 0; i-- {
				if !yield(items[i]) {
					return
				}
			}
		},
		capacity: q.capacity,
		materialize: func() []T {
			if q.fastSlice != nil {
				if q.fastWhere == nil {
					n := len(q.fastSlice)
					result := make([]T, n)
					for i := 0; i < n; i++ {
						result[i] = q.fastSlice[n-1-i]
					}
					return result
				}
				predicate := q.fastWhere
				filtered := make([]T, 0, q.capacity/2+1)
				for _, item := range q.fastSlice {
					if predicate(item) {
						filtered = append(filtered, item)
					}
				}
				for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
				return filtered
			}
			var result []T
			if q.capacity > 0 {
				result = make([]T, 0, q.capacity)
			}
			for item := range q.iterate {
				result = append(result, item)
			}
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
			return result
		},
	}
}

// Distinct 代理
func (q Query[T]) Distinct() Query[T] {
	return Distinct(q)
}

// Intersect 代理
func (q Query[T]) Intersect(q2 Query[T]) Query[T] {
	return Intersect(q, q2)
}

// Union 代理
func (q Query[T]) Union(q2 Query[T]) Query[T] {
	return Union(q, q2)
}

// Except 代理
func (q Query[T]) Except(q2 Query[T]) Query[T] {
	return Except(q, q2)
}

// AppendTo 追加到目标切片中
func (q Query[T]) AppendTo(dest []T) []T {
	if q.capacity > 0 {
		dest = slices.Grow(dest, q.capacity)
	}
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			dest = append(dest, item)
		}
		return dest
	}
	for item := range q.iterate {
		dest = append(dest, item)
	}
	return dest
}

// ToMapSlice 将序列转换为 []map[string]T，通常用于 JSON 序列化
func (q Query[T]) ToMapSlice(selector func(T) map[string]T) (r []map[string]T) {
	if q.capacity > 0 {
		r = make([]map[string]T, 0, q.capacity)
	}
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			r = append(r, selector(item))
		}
		return r
	}
	for item := range q.iterate {
		r = append(r, selector(item))
	}
	return r
}

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
func (q Query[T]) ForEachParallelCtx(ctx context.Context, action func(T), workers ...int) {
	if ctx == nil {
		ctx = context.Background()
	}
	iworkers := 1
	if len(workers) > 0 && workers[0] > 0 {
		iworkers = workers[0]
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan T, iworkers)
	errCh := make(chan any, 1)
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
					action(item)
				}
			}
		}()
	}

	emit := func(item T) bool {
		select {
		case <-workerCtx.Done():
			return false
		case jobs <- item:
			return true
		}
	}

	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			if !emit(item) {
				break
			}
		}
	} else {
		for item := range q.iterate {
			if !emit(item) {
				break
			}
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case panicErr := <-errCh:
		panic(panicErr)
	default:
	}
}

// ForEachParallel 并发遍历无 context (底层封装 Ctx 版本)
func (q Query[T]) ForEachParallel(action func(T), workers ...int) {
	q.ForEachParallelCtx(context.Background(), action, workers...)
}

// Count 返回序列中的元素个数
func (q Query[T]) Count() int {
	if q.fastSlice != nil {
		if q.fastWhere == nil {
			return len(q.fastSlice)
		}
		count := 0
		for _, item := range q.fastSlice {
			if q.fastWhere(item) {
				count++
			}
		}
		return count
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
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			if predicate(v) {
				return true
			}
		}
		return false
	}
	for item := range q.iterate {
		if predicate(item) {
			return true
		}
	}
	return false
}

// All 判断序列中的所有元素是否都满足指定条件
func (q Query[T]) All(predicate func(T) bool) bool {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			if !predicate(v) {
				return false
			}
		}
		return true
	}
	for item := range q.iterate {
		if !predicate(item) {
			return false
		}
	}
	return true
}

func (q Query[T]) SumIntBy(selector func(T) int) int             { return SumBy(q, selector) }
func (q Query[T]) SumInt8By(selector func(T) int8) int8          { return SumBy(q, selector) }
func (q Query[T]) SumInt16By(selector func(T) int16) int16       { return SumBy(q, selector) }
func (q Query[T]) SumInt32By(selector func(T) int32) int32       { return SumBy(q, selector) }
func (q Query[T]) SumInt64By(selector func(T) int64) int64       { return SumBy(q, selector) }
func (q Query[T]) SumUIntBy(selector func(T) uint) uint          { return SumBy(q, selector) }
func (q Query[T]) SumUInt8By(selector func(T) uint8) uint8       { return SumBy(q, selector) }
func (q Query[T]) SumUInt16By(selector func(T) uint16) uint16    { return SumBy(q, selector) }
func (q Query[T]) SumUInt32By(selector func(T) uint32) uint32    { return SumBy(q, selector) }
func (q Query[T]) SumUInt64By(selector func(T) uint64) uint64    { return SumBy(q, selector) }
func (q Query[T]) SumFloat32By(selector func(T) float32) float32 { return SumBy(q, selector) }
func (q Query[T]) SumFloat64By(selector func(T) float64) float64 { return SumBy(q, selector) }
func (q Query[T]) AvgBy(selector func(T) float64) float64        { return AverageBy(q, selector) }
func (q Query[T]) AvgIntBy(selector func(T) int) float64         { return AverageBy(q, selector) }
func (q Query[T]) AvgInt64By(selector func(T) int64) float64     { return AverageBy(q, selector) }

// First 返回第一元素，如果没有则返回零值
func (q Query[T]) First() T {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			return v
		}
		var zero T
		return zero
	}
	for item := range q.iterate {
		return item
	}
	var zero T
	return zero
}

// FirstWith 返回满足条件的第一个元素
func (q Query[T]) FirstWith(predicate func(T) bool) T {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			if predicate(v) {
				return v
			}
		}
		var zero T
		return zero
	}
	for item := range q.iterate {
		if predicate(item) {
			return item
		}
	}
	var zero T
	return zero
}

// FirstOK 返回第一个元素以及是否存在
func (q Query[T]) FirstOK() (T, bool) {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			return v, true
		}
		var zero T
		return zero, false
	}
	for item := range q.iterate {
		return item, true
	}
	var zero T
	return zero, false
}

// FirstWithOK 返回满足条件的第一个元素以及是否存在
func (q Query[T]) FirstWithOK(predicate func(T) bool) (T, bool) {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			if predicate(v) {
				return v, true
			}
		}
		var zero T
		return zero, false
	}
	for item := range q.iterate {
		if predicate(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
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
			var zero T
			return zero
		} else {
			for i := len(source) - 1; i >= 0; i-- {
				if pre(source[i]) {
					return source[i]
				}
			}
			var zero T
			return zero
		}
	}
	var last T
	for item := range q.iterate {
		last = item
	}
	return last
}

// LastOK 返回最后一个元素以及是否存在
func (q Query[T]) LastOK() (T, bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		pre := q.fastWhere
		if pre == nil {
			if len(source) > 0 {
				return source[len(source)-1], true
			}
		} else {
			for i := len(source) - 1; i >= 0; i-- {
				if pre(source[i]) {
					return source[i], true
				}
			}
		}
		var zero T
		return zero, false
	}
	var last T
	found := false
	for item := range q.iterate {
		last = item
		found = true
	}
	return last, found
}

// LastWith 返回满足条件的最后一个元素
func (q Query[T]) LastWith(predicate func(T) bool) T {
	if q.fastSlice != nil {
		source := q.fastSlice
		pre := q.fastWhere
		for i := len(source) - 1; i >= 0; i-- {
			v := source[i]
			if pre != nil && !pre(v) {
				continue
			}
			if predicate(v) {
				return v
			}
		}
		var zero T
		return zero
	}
	var last T
	for item := range q.iterate {
		if predicate(item) {
			last = item
		}
	}
	return last
}

// LastWithOK 返回满足条件的最后一个元素以及是否存在
func (q Query[T]) LastWithOK(predicate func(T) bool) (T, bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		pre := q.fastWhere
		for i := len(source) - 1; i >= 0; i-- {
			v := source[i]
			if pre != nil && !pre(v) {
				continue
			}
			if predicate(v) {
				return v, true
			}
		}
		var zero T
		return zero, false
	}
	var last T
	found := false
	for item := range q.iterate {
		if predicate(item) {
			last = item
			found = true
		}
	}
	return last, found
}

// FirstDefault 返回第一个元素，若空返回 defaultValue
func (q Query[T]) FirstDefault(defaultValue ...T) T {
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			return v
		}
	} else {
		for item := range q.iterate {
			return item
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	var zero T
	return zero
}

// LastDefault 返回最后一个元素，若空返回 defaultValue
func (q Query[T]) LastDefault(defaultValue ...T) T {
	if q.fastSlice != nil {
		source := q.fastSlice
		pre := q.fastWhere
		for i := len(source) - 1; i >= 0; i-- {
			v := source[i]
			if pre != nil && !pre(v) {
				continue
			}
			return v
		}
	} else {
		var last T
		found := false
		for item := range q.iterate {
			last = item
			found = true
		}
		if found {
			return last
		}
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
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			val = v
			count++
			if count > 1 {
				var zero T
				return zero
			}
		}
	} else {
		for item := range q.iterate {
			val = item
			count++
			if count > 1 {
				var zero T
				return zero
			}
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
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			val = v
			count++
			if count > 1 {
				if len(defaultValue) > 0 {
					return defaultValue[0]
				}
				var zero T
				return zero
			}
		}
	} else {
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

// SingleOK 返回唯一元素以及是否存在且唯一
func (q Query[T]) SingleOK() (T, bool) {
	var val T
	count := 0
	if q.fastSlice != nil {
		for _, v := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(v) {
				continue
			}
			val = v
			count++
			if count > 1 {
				var zero T
				return zero, false
			}
		}
	} else {
		for item := range q.iterate {
			val = item
			count++
			if count > 1 {
				var zero T
				return zero, false
			}
		}
	}
	if count == 1 {
		return val, true
	}
	var zero T
	return zero, false
}

// SingleWithOK 返回满足条件的唯一元素以及是否存在且唯一
func (q Query[T]) SingleWithOK(predicate func(T) bool) (T, bool) {
	return q.Where(predicate).SingleOK()
}

// IndexOfWith 返回满足条件的元素的索引
func (q Query[T]) IndexOfWith(predicate func(T) bool) int {
	index := 0
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			if predicate(item) {
				return index
			}
			index++
		}
	} else {
		for item := range q.iterate {
			if predicate(item) {
				return index
			}
			index++
		}
	}
	return -1
}

// LastIndexOfWith 返回满足条件的元素最后出现的索引
func (q Query[T]) LastIndexOfWith(predicate func(T) bool) int {
	index := 0
	last := -1
	if q.fastSlice != nil {
		for _, item := range q.fastSlice {
			if q.fastWhere != nil && !q.fastWhere(item) {
				continue
			}
			if predicate(item) {
				last = index
			}
			index++
		}
	} else {
		for item := range q.iterate {
			if predicate(item) {
				last = index
			}
			index++
		}
	}
	return last
}

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
			iterate:   q.iterate,
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
			return QueryEmpty[T]()
		}
		if count <= 0 {
			return q
		}
		return From(q.fastSlice[count:])
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			n := count
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					if n > 0 {
						n--
						continue
					}
					if !yield(item) {
						break
					}
				}
				return
			}
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
			return QueryEmpty[T]()
		}
		if count >= len(q.fastSlice) {
			return q
		}
		return From(q.fastSlice[:count])
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			n := count
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					if n <= 0 {
						break
					}
					n--
					if !yield(item) {
						break
					}
				}
				return
			}
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
		capacity: q.capacity + q2.capacity,
		materialize: func() []T {
			result := make([]T, 0, q.capacity+q2.capacity)
			if q.fastSlice != nil {
				if q.fastWhere == nil {
					result = append(result, q.fastSlice...)
				} else {
					for _, t := range q.fastSlice {
						if q.fastWhere(t) {
							result = append(result, t)
						}
					}
				}
			} else {
				for t := range q.iterate {
					result = append(result, t)
				}
			}

			if q2.fastSlice != nil {
				if q2.fastWhere == nil {
					result = append(result, q2.fastSlice...)
				} else {
					for _, t := range q2.fastSlice {
						if q2.fastWhere(t) {
							result = append(result, t)
						}
					}
				}
			} else {
				for t := range q2.iterate {
					result = append(result, t)
				}
			}
			return result
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
			if q.fastSlice != nil {
				for _, item := range q.fastSlice {
					if q.fastWhere != nil && !q.fastWhere(item) {
						continue
					}
					empty = false
					if !yield(item) {
						return
					}
				}
			} else {
				for item := range q.iterate {
					empty = false
					if !yield(item) {
						return
					}
				}
			}
			if empty {
				yield(defaultValue)
			}
		},
	}
}
