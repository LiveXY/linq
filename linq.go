package linq

import (
	"cmp"
	"context"
	"math/rand/v2"
	"sort"
	"sync"
	"time"
	"unicode/utf8"
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

// HasOrder 判断查询目前是否已定义排序规则
func (q Query[T]) HasOrder() bool {
	return q.lesser != nil
}

// OrderBy 指定主要排序键，按升序对序列元素进行排序
func OrderBy[T comparable, K cmp.Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	})
}

// OrderByDescending 指定主要排序键，按降序对序列元素进行排序
func OrderByDescending[T comparable, K cmp.Ordered](q Query[T], key func(t T) K) Query[T] {
	return orderByLesser(q, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) > key(data[j])
		}
	})
}

// ThenBy 指定次要排序键，按升序对序列元素进行后续排序
// 必须在 OrderBy 或 OrderByDescending 之后调用
func ThenBy[T comparable, K cmp.Ordered](q Query[T], key func(t T) K) Query[T] {
	lesser := q.lesser
	return orderByLesser(q, chainLessers(lesser, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) < key(data[j])
		}
	}))
}

// ThenByDescending 指定次要排序键，按降序对序列元素进行后续排序
// 必须在 OrderBy 或 OrderByDescending 之后调用
func ThenByDescending[T comparable, K cmp.Ordered](q Query[T], key func(t T) K) Query[T] {
	lesser := q.lesser
	return orderByLesser(q, chainLessers(lesser, func(data []T) func(i, j int) bool {
		return func(i, j int) bool {
			return key(data[i]) > key(data[j])
		}
	}))
}

func chainLessers[T comparable](a, b lesserFunc[T]) lesserFunc[T] {
	return func(data []T) func(i, j int) bool {
		a, b := a(data), b(data)
		return func(i, j int) bool {
			return a(i, j) || !a(j, i) && b(i, j)
		}
	}
}
func orderByLesser[T comparable](q Query[T], lesser lesserFunc[T]) Query[T] {
	return Query[T]{
		lesser: lesser,
		iterate: func() func() (T, bool) {
			data := q.ToSlice()
			sort.Slice(data, lesser(data))
			return From(data).iterate()
		},
		capacity: q.capacity,
	}
}

type lesserFunc[T comparable] func([]T) func(i, j int) bool

// KV 键值对结构体，用于存储分组等操作的结果
type KV[K, V comparable] struct {
	Key   K
	Value V
}

// Query 查询结构体，是 LINQ 操作的核心类型
type Query[T comparable] struct {
	lesser  lesserFunc[T]
	iterate func() func() (T, bool)

	// fastSlice 和 fastWhere 用于优化切片操作
	// 当源是切片且仅进行了 Where 操作时，ToSlice 可以直接循环而不是通过迭代器
	fastSlice []T
	fastWhere func(T) bool

	// capacity 用于预估结果集大小，以便在 ToSlice 等操作中进行内存预分配
	capacity int
}

// From 从切片创建 Query 查询对象
func From[T comparable](source []T) Query[T] {
	length := len(source)
	return Query[T]{
		iterate: func() func() (T, bool) {
			index := 0
			return func() (item T, ok bool) {
				ok = index < length
				if ok {
					item = source[index]
					index++
				}
				return
			}
		},
		fastSlice: source,
		capacity:  length,
	}
}

// FromChannel 从只读 Channel 创建 Query 查询对象
func FromChannel[T comparable](source <-chan T) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			return func() (item T, ok bool) {
				item, ok = <-source
				return
			}
		},
	}
}

// FromString 从字符串创建 Query 查询对象，每个元素为一个 UTF-8 字符
func FromString(source string) Query[string] {
	return Query[string]{
		iterate: func() func() (string, bool) {
			pos := 0
			length := len(source)
			return func() (item string, ok bool) {
				if pos >= length {
					return
				}
				r, w := utf8.DecodeRuneInString(source[pos:])
				if r == utf8.RuneError && w == 1 {
					item = string(r)
				} else {
					item = source[pos : pos+w]
				}
				pos += w
				ok = true
				return
			}
		},
	}
}

// FromMap 从 Map 创建 Query 查询对象，每个元素为 KV 键值对
func FromMap[K, V comparable](source map[K]V) Query[KV[K, V]] {
	length := len(source)
	keyvalues := make([](KV[K, V]), 0, length)
	for key, value := range source {
		keyvalues = append(keyvalues, KV[K, V]{Key: key, Value: value})
	}
	return From(keyvalues)
}

// Where 返回满足指定条件的元素序列
func (q Query[T]) Where(predicate func(T) bool) Query[T] {
	// 如果由于源是切片（fastSlice），我们可以进行 "Iterator Flattening"（迭代器扁平化）。
	// 我们不再包裹上游的迭代器，而是直接基于原始切片构建一个新的迭代器，并组合过滤条件。
	// 这使得无论有多少个 Where 链式调用，迭代器层级永远只有一层，极大减少了函数调用开销。
	if q.fastSlice != nil {
		source := q.fastSlice

		// 组合过滤条件
		var combinedPred func(T) bool
		if q.fastWhere == nil {
			combinedPred = predicate
		} else {
			oldPred := q.fastWhere
			combinedPred = func(t T) bool {
				return oldPred(t) && predicate(t)
			}
		}

		return Query[T]{
			// 构建扁平化的迭代器
			iterate: func() func() (T, bool) {
				index := 0
				length := len(source)
				return func() (item T, ok bool) {
					for index < length {
						item = source[index]
						index++
						if combinedPred(item) {
							return item, true
						}
					}
					return
				}
			},
			fastSlice: source,
			fastWhere: combinedPred,
		}
	}

	// 传统的装饰器模式，包裹上游迭代器
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if predicate(item) {
						return
					}
				}
				return
			}
		},
		capacity: q.capacity,
	}
}

// Skip 跳过前 N 个元素
func (q Query[T]) Skip(count int) Query[T] {
	// 纯切片操作，直接调整切片窗口
	if q.fastSlice != nil && q.fastWhere == nil {
		if count >= len(q.fastSlice) {
			return From([]T{})
		}
		if count <= 0 {
			return q
		}
		return From(q.fastSlice[count:])
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			n := count
			return func() (item T, ok bool) {
				for ; n > 0; n-- {
					item, ok = next()
					if !ok {
						return
					}
				}
				return next()
			}
		},
	}
}

// Take 获取前 N 个元素
func (q Query[T]) Take(count int) Query[T] {
	// 纯切片操作，直接调整切片窗口
	if q.fastSlice != nil && q.fastWhere == nil {
		if count <= 0 {
			return From([]T{})
		}
		if count >= len(q.fastSlice) {
			return q
		}
		return From(q.fastSlice[:count])
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			n := count
			return func() (item T, ok bool) {
				if n <= 0 {
					return
				}
				n--
				return next()
			}
		},
	}
}

// TakeWhile 获取满足条件的元素，直到遇到不满足条件的元素
func (q Query[T]) TakeWhile(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere // May be nil

		iterator := func() func() (T, bool) {
			index := 0
			length := len(source)
			active := true
			return func() (item T, ok bool) {
				if !active {
					return
				}
				for index < length {
					item = source[index]
					index++

					if preFilter != nil {
						if !preFilter(item) {
							continue
						}
					}

					if !predicate(item) {
						active = false
						ok = false
						return
					}

					return item, true
				}
				return
			}
		}
		return Query[T]{
			iterate: iterator,
		}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			taking := true
			return func() (item T, ok bool) {
				if !taking {
					return
				}
				item, ok = next()
				if !ok {
					taking = false
					return
				}
				if !predicate(item) {
					taking = false
					ok = false
					return
				}
				return item, true
			}
		},
	}
}

// SkipWhile 跳过满足条件的元素，直到遇到不满足条件的元素
func (q Query[T]) SkipWhile(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere

		iterator := func() func() (T, bool) {
			index := 0
			length := len(source)
			skipping := true
			return func() (item T, ok bool) {
				for index < length {
					item = source[index]
					index++

					if preFilter != nil {
						if !preFilter(item) {
							continue
						}
					}

					if skipping {
						if predicate(item) {
							continue
						}
						skipping = false
						return item, true
					}

					return item, true
				}
				return
			}
		}
		return Query[T]{
			iterate: iterator,
		}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			skipping := true
			return func() (item T, ok bool) {
				for skipping {
					item, ok = next()
					if !ok {
						return
					}
					if !predicate(item) {
						skipping = false
						return item, true
					}
				}
				return next()
			}
		},
	}
}

// Page 分页查询，返回指定页码和页大小的元素
func (q Query[T]) Page(page, pageSize int) Query[T] {
	return q.Skip((page - 1) * pageSize).Take(pageSize)
}

// Union 返回两个序列的并集，自动去重
func (q Query[T]) Union(q2 Query[T]) Query[T] {
	// 如果两个都是纯切片，直接做 Map 处理
	if q.fastSlice != nil && q.fastWhere == nil && q2.fastSlice != nil && q2.fastWhere == nil {
		// 返回一个新的 Query，这个 Query 本身又是一个基于 slice 的 fast path query
		// 但为了保持懒加载语义，我们得把计算包在一个闭包里？
		// 不，为了性能，我们可以在 iterate 首次调用时一次性计算好，或者分步输出。
		// 最好的方式是扁平化输出。

		s1 := q.fastSlice
		s2 := q2.fastSlice

		return Query[T]{
			iterate: func() func() (T, bool) {
				idx1 := 0
				idx2 := 0
				len1 := len(s1)
				len2 := len(s2)
				set := make(map[T]struct{}, len1+len2) // 预分配

				return func() (item T, ok bool) {
					// 遍历第一个 slice
					for idx1 < len1 {
						item = s1[idx1]
						idx1++
						if _, has := set[item]; !has {
							set[item] = struct{}{}
							return item, true
						}
					}
					// 遍历第二个 slice
					for idx2 < len2 {
						item = s2[idx2]
						idx2++
						if _, has := set[item]; !has {
							set[item] = struct{}{}
							return item, true
						}
					}
					return
				}
			},
		}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[T]struct{})
			use1 := true
			return func() (item T, ok bool) {
				if use1 {
					for item, ok = next(); ok; item, ok = next() {
						if _, has := set[item]; !has {
							set[item] = struct{}{}
							return
						}
					}
					use1 = false
				}
				for item, ok = next2(); ok; item, ok = next2() {
					if _, has := set[item]; !has {
						set[item] = struct{}{}
						return
					}
				}
				return
			}
		},
	}
}

// Append 在序列末尾追加一个元素
func (q Query[T]) Append(item T) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere

		iterator := func() func() (T, bool) {
			index := 0
			length := len(source)
			appended := false
			return func() (T, bool) {
				for index < length {
					t := source[index]
					index++
					if predicate != nil && !predicate(t) {
						continue
					}
					return t, true
				}
				if !appended {
					appended = true
					return item, true
				}
				var zero T
				return zero, false
			}
		}
		return Query[T]{iterate: iterator}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			appended := false
			var t T
			return func() (T, bool) {
				i, ok := next()
				if ok {
					return i, ok
				}
				if !appended {
					appended = true
					return item, true
				}
				return t, false
			}
		},
	}
}

// Concat 连接两个序列
func (q Query[T]) Concat(q2 Query[T]) Query[T] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere

		iterator := func() func() (T, bool) {
			idx1 := 0
			len1 := len(s1)
			idx2 := 0
			len2 := len(s2)
			use1 := true

			return func() (item T, ok bool) {
				if use1 {
					for idx1 < len1 {
						item = s1[idx1]
						idx1++
						if p1 != nil && !p1(item) {
							continue
						}
						return item, true
					}
					use1 = false
				}
				for idx2 < len2 {
					item = s2[idx2]
					idx2++
					if p2 != nil && !p2(item) {
						continue
					}
					return item, true
				}
				return
			}
		}
		return Query[T]{iterate: iterator}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			use1 := true
			return func() (item T, ok bool) {
				if use1 {
					item, ok = next()
					if ok {
						return
					}
					use1 = false
				}
				return next2()
			}
		},
	}
}

// Prepend 在序列开头插入一个元素
func (q Query[T]) Prepend(item T) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere

		iterator := func() func() (T, bool) {
			index := 0
			length := len(source)
			prepended := false
			return func() (T, bool) {
				// 1. Prepend item
				if !prepended {
					prepended = true
					return item, true
				}
				// 2. Iterate over fastSlice
				for index < length {
					t := source[index]
					index++
					if predicate != nil && !predicate(t) {
						continue
					}
					return t, true
				}
				var zero T
				return zero, false
			}
		}
		return Query[T]{iterate: iterator}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			prepended := false
			return func() (T, bool) {
				if prepended {
					return next()
				}
				prepended = true
				return item, true
			}
		},
	}
}

// DefaultIfEmpty 如果序列为空，返回包含默认值的序列
func (q Query[T]) DefaultIfEmpty(defaultValue T) Query[T] {
	if q.fastSlice != nil && q.fastWhere == nil {
		if len(q.fastSlice) == 0 {
			return From([]T{defaultValue})
		}
		return q
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			state := 1
			return func() (item T, ok bool) {
				switch state {
				case 1:
					item, ok = next()
					if ok {
						state = 2
					} else {
						item = defaultValue
						ok = true
						state = -1
					}
					return
				case 2:
					for item, ok = next(); ok; item, ok = next() {
						return
					}
					return
				}
				return
			}
		},
	}
}

// Distinct 返回去重后的序列
func (q Query[T]) Distinct() Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			set := make(map[T]struct{})
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if _, has := set[item]; !has {
						set[item] = struct{}{}
						return
					}
				}
				return
			}
		},
	}
}

// Except 返回差集，即在第一个序列中但不在第二个序列中的元素
func (q Query[T]) Except(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[T]struct{})
			for i, ok := next2(); ok; i, ok = next2() {
				set[i] = struct{}{}
			}
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if _, has := set[item]; !has {
						return
					}
				}
				return
			}
		},
	}
}

// IndexOf 返回第一个满足条件的元素索引，未找到返回 -1
func (q Query[T]) IndexOf(predicate func(T) bool) int {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere

		if preFilter == nil {
			for i, item := range source {
				if predicate(item) {
					return i
				}
			}
			return -1
		}

		logicalIndex := 0
		for _, item := range source {
			if !preFilter(item) {
				continue
			}
			if predicate(item) {
				return logicalIndex
			}
			logicalIndex++
		}
		return -1
	}

	index := 0
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			return index
		}
		index++
	}
	return -1
}

// Intersect 返回交集，即同时存在于两个序列中的元素
func (q Query[T]) Intersect(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[T]struct{})
			for item, ok := next2(); ok; item, ok = next2() {
				set[item] = struct{}{}
			}
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if _, has := set[item]; has {
						delete(set, item)
						return
					}
				}
				return
			}
		},
	}
}

// All 判断是否所有元素都满足指定条件
func (q Query[T]) All(predicate func(T) bool) bool {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if !predicate(item) {
				return false
			}
		}
		return true
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// Any 判断序列是否包含任何元素
func (q Query[T]) Any() bool {
	if q.fastSlice != nil {
		if q.fastWhere == nil {
			return len(q.fastSlice) > 0
		}
		for _, item := range q.fastSlice {
			if q.fastWhere(item) {
				return true
			}
		}
		return false
	}

	_, ok := q.iterate()()
	return ok
}

// AnyWith 判断是否存在满足条件的元素
func (q Query[T]) AnyWith(predicate func(T) bool) bool {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if predicate(item) {
				return true
			}
		}
		return false
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			return true
		}
	}
	return false
}

// CountWith 返回满足条件的元素数量
func (q Query[T]) CountWith(predicate func(T) bool) (r int) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if predicate(item) {
				r++
			}
		}
		return
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			r++
		}
	}
	return
}

// First 返回序列的第一个元素
func (q Query[T]) First() T {
	if q.fastSlice != nil {
		if q.fastWhere == nil {
			if len(q.fastSlice) > 0 {
				return q.fastSlice[0]
			}
			var zero T
			return zero
		}
		for _, item := range q.fastSlice {
			if q.fastWhere(item) {
				return item
			}
		}
		var zero T
		return zero
	}

	item, _ := q.iterate()()
	return item
}

// FirstWith 返回第一个满足条件的元素
func (q Query[T]) FirstWith(predicate func(T) bool) T {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if predicate(item) {
				return item
			}
		}
		var zero T
		return zero
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			return item
		}
	}
	var out T
	return out
}

// ForEach 遍历序列中的每个元素，返回 false 可提前终止
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

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !action(item) {
			return
		}
	}
}

// ForEachIndexed 带索引遍历序列中的每个元素
func (q Query[T]) ForEachIndexed(action func(int, T) bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		index := 0
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

	next := q.iterate()
	index := 0
	for item, ok := next(); ok; item, ok = next() {
		if !action(index, item) {
			return
		}
		index++
	}
}

// ForEachParallel 并发遍历序列中的元素，指定工作线程数
func (q Query[T]) ForEachParallel(workers int, action func(T)) {
	if workers <= 1 {
		q.ForEach(func(t T) bool {
			action(t)
			return true
		})
		return
	}

	ch := make(chan T, workers)
	var wg sync.WaitGroup // Requires "sync"
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// 记录 panic 但不中断其他 worker
					// 在生产环境中应该使用日志记录
					_ = r
				}
			}()
			for item := range ch {
				action(item)
			}
		}()
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		ch <- item
	}
	close(ch)
	wg.Wait()
}

// ForEachParallelCtx 并发遍历序列中的元素，支持 Context 取消
func (q Query[T]) ForEachParallelCtx(ctx context.Context, workers int, action func(T)) {
	if workers <= 1 {
		q.ForEach(func(t T) bool {
			select {
			case <-ctx.Done():
				return false
			default:
				action(t)
				return true
			}
		})
		return
	}

	ch := make(chan T, workers)
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					_ = r
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-ch:
					if !ok {
						return
					}
					action(item)
				}
			}
		}()
	}

	next := q.iterate()
Loop:
	for item, ok := next(); ok; item, ok = next() {
		select {
		case <-ctx.Done():
			break Loop
		case ch <- item:
		}
	}
	close(ch)
	wg.Wait()
}

// Last 返回序列的最后一个元素
func (q Query[T]) Last() (r T) {
	if q.fastSlice != nil && q.fastWhere == nil {
		if len(q.fastSlice) > 0 {
			return q.fastSlice[len(q.fastSlice)-1]
		}
		return
	}
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		for i := len(source) - 1; i >= 0; i-- {
			item := source[i]
			if predicate(item) {
				return item
			}
		}
		return
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = item
	}
	return
}

// LastWith 返回最后一个满足条件的元素
func (q Query[T]) LastWith(predicate func(T) bool) (r T) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for i := len(source) - 1; i >= 0; i-- {
			item := source[i]
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if predicate(item) {
				return item
			}
		}
		return
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			r = item
		}
	}
	return
}

// Reverse 返回反转后的序列
func (q Query[T]) Reverse() Query[T] {
	if q.fastSlice != nil && q.fastWhere == nil {
		return Query[T]{
			iterate: func() func() (T, bool) {
				index := len(q.fastSlice) - 1
				return func() (item T, ok bool) {
					if index < 0 {
						return
					}
					item = q.fastSlice[index]
					index--
					return item, true
				}
			},
			capacity: len(q.fastSlice),
		}
	}

	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			var items []T
			if q.capacity > 0 {
				items = make([]T, 0, q.capacity)
			}
			for item, ok := next(); ok; item, ok = next() {
				items = append(items, item)
			}
			index := len(items) - 1
			return func() (item T, ok bool) {
				if index < 0 {
					return
				}
				item, ok = items[index], true
				index--
				return
			}
		},
		capacity: q.capacity,
	}
}

// Single 返回序列中的唯一元素，如果序列为空或包含多个元素则返回零值
func (q Query[T]) Single() (r T) {
	next := q.iterate()
	item, ok := next()
	if !ok {
		return r
	}
	_, ok = next()
	if ok {
		return r
	}
	return item
}

// SumInt8By 计算序列中 int8 属性的总和
func (q Query[T]) SumInt8By(selector func(T) int8) int8 {
	return SumBy(q, selector)
}

// SumInt16By 计算序列中 int16 属性的总和
func (q Query[T]) SumInt16By(selector func(T) int16) int16 {
	return SumBy(q, selector)
}

// SumIntBy 计算序列中 int 属性的总和
func (q Query[T]) SumIntBy(selector func(T) int) int {
	return SumBy(q, selector)
}

// SumInt32By 计算序列中 int32 属性的总和
func (q Query[T]) SumInt32By(selector func(T) int32) int32 {
	return SumBy(q, selector)
}

// SumInt64By 计算序列中 int64 属性的总和
func (q Query[T]) SumInt64By(selector func(T) int64) int64 {
	return SumBy(q, selector)
}

// SumUInt8By 计算序列中 uint8 属性的总和
func (q Query[T]) SumUInt8By(selector func(T) uint8) uint8 {
	return SumBy(q, selector)
}

// SumUInt16By 计算序列中 uint16 属性的总和
func (q Query[T]) SumUInt16By(selector func(T) uint16) uint16 {
	return SumBy(q, selector)
}

// SumUIntBy 计算序列中 uint 属性的总和
func (q Query[T]) SumUIntBy(selector func(T) uint) uint {
	return SumBy(q, selector)
}

// SumUInt32By 计算序列中 uint32 属性的总和
func (q Query[T]) SumUInt32By(selector func(T) uint32) uint32 {
	return SumBy(q, selector)
}

// SumUInt64By 计算序列中 uint64 属性的总和
func (q Query[T]) SumUInt64By(selector func(T) uint64) uint64 {
	return SumBy(q, selector)
}

// SumFloat32By 计算序列中 float32 属性的总和
func (q Query[T]) SumFloat32By(selector func(T) float32) float32 {
	return SumBy(q, selector)
}

// SumFloat64By 计算序列中 float64 属性的总和
func (q Query[T]) SumFloat64By(selector func(T) float64) float64 {
	return SumBy(q, selector)
}

// AvgIntBy 计算序列中 int 属性的平均值
func (q Query[T]) AvgIntBy(selector func(T) int) float64 {
	return AvgBy(q, selector)
}

// AvgInt64By 计算序列中 int64 属性的平均值
func (q Query[T]) AvgInt64By(selector func(T) int64) float64 {
	return AvgBy(q, selector)
}

// AvgBy 计算序列中 float64 属性的平均值（兼容方法，内部调用泛型 AvgBy 函数）
func (q Query[T]) AvgBy(selector func(T) float64) float64 {
	return AvgBy(q, selector)
}

// Count 返回序列中的元素数量
func (q Query[T]) Count() (r int) {
	// 如果是纯切片，直接返回长度，O(1) 复杂度！
	if q.fastSlice != nil && q.fastWhere == nil {
		return len(q.fastSlice)
	}
	// 如果包含过滤条件，也可以优化循环，但必须遍历
	if q.fastSlice != nil && q.fastWhere != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		for _, item := range source {
			if predicate(item) {
				r++
			}
		}
		return
	}

	next := q.iterate()
	for _, ok := next(); ok; _, ok = next() {
		r++
	}
	return
}

// ToSlice 将序列转换为切片
func (q Query[T]) ToSlice() (r []T) {
	// 如果源是切片且只有过滤条件，直接循环，避免迭代器开销
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		// 优先使用实际切片长度作为容量，如果存在 fastSlice，这通常是最准确的上限
		r = make([]T, 0, len(source))

		if predicate == nil {
			r = append(r, source...)
			return
		}

		for _, item := range source {
			if predicate(item) {
				r = append(r, item)
			}
		}
		return
	}

	// 使用 capacity 进行预分配
	if q.capacity > 0 {
		r = make([]T, 0, q.capacity)
	}

	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, item)
	}
	return
}

// AppendTo 将序列中的元素追加到指定的切片中
func (q Query[T]) AppendTo(dest []T) []T {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		dest = append(dest, item)
	}
	return dest
}

// ToChannel 将序列写入到指定的只写 Channel
func (q Query[T]) ToChannel(c chan<- T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		c <- item
	}
	close(c)
}

// ToMapSlice 将序列转换为 []map[string]T，通常用于 JSON 序列化
func (q Query[T]) ToMapSlice(selector func(T) map[string]T) (r []map[string]T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, selector(item))
	}
	return
}

// GroupBy 根据 keySelector 对序列进行分组
func GroupBy[T comparable, K comparable](q Query[T], keySelector func(T) K) Query[KV[K, *[]T]] {
	// 优化：如果源是切片，进行两遍扫描以减少内存分配
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere

		return Query[KV[K, *[]T]]{
			iterate: func() func() (KV[K, *[]T], bool) {
				// 第一遍扫描：计算每组的大小，并记录键的出现顺序
				counts := make(map[K]int) // TODO: 可以在 Query 中添加 KeyCount hint? 不，太复杂。
				var order []K
				// 预估 order 容量，避免频繁扩容。假设平均每组 4 个元素？或者是 sqrt(len)?
				// 既然无法预知，还是默认吧，或者小一点。

				for _, item := range source {
					if predicate != nil && !predicate(item) {
						continue
					}
					key := keySelector(item)
					if counts[key] == 0 {
						order = append(order, key)
					}
					counts[key]++
				}

				// 预分配 Map 和 Slices
				set := make(map[K][]T, len(counts))
				for _, key := range order {
					set[key] = make([]T, 0, counts[key])
				}

				// 第二遍扫描：填充数据
				for _, item := range source {
					if predicate != nil && !predicate(item) {
						continue
					}
					key := keySelector(item)
					set[key] = append(set[key], item)
				}

				length := len(order)
				index := 0
				return func() (item KV[K, *[]T], ok bool) {
					ok = index < length
					if ok {
						key := order[index]
						slice := set[key]
						item = KV[K, *[]T]{key, &slice}
						index++
					}
					return
				}
			},
		}
	}

	return Query[KV[K, *[]T]]{
		iterate: func() func() (KV[K, *[]T], bool) {
			next := q.iterate()
			set := make(map[K][]T)
			var keys []K
			for item, ok := next(); ok; item, ok = next() {
				key := keySelector(item)
				if _, ok := set[key]; !ok {
					keys = append(keys, key)
				}
				set[key] = append(set[key], item)
			}
			length := len(keys)
			index := 0
			return func() (item KV[K, *[]T], ok bool) {
				ok = index < length
				if ok {
					key := keys[index]
					// 返回切片的指针以满足 comparable 约束
					slice := set[key]
					item = KV[K, *[]T]{key, &slice}
					index++
				}
				return
			}
		},
	}
}

// GroupBySelect 根据 keySelector 分组，并对元素应用 elementSelector
func GroupBySelect[T, K, V comparable](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[KV[K, *[]V]] {
	// 优化：如果源是切片，进行两遍扫描
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere

		return Query[KV[K, *[]V]]{
			iterate: func() func() (KV[K, *[]V], bool) {
				// Pass 1: Count
				counts := make(map[K]int)
				var order []K

				for _, item := range source {
					if predicate != nil && !predicate(item) {
						continue
					}
					key := keySelector(item)
					if counts[key] == 0 {
						order = append(order, key)
					}
					counts[key]++
				}

				// Pre-allocate
				set := make(map[K][]V, len(counts))
				for _, key := range order {
					set[key] = make([]V, 0, counts[key])
				}

				// Pass 2: Fill
				for _, item := range source {
					if predicate != nil && !predicate(item) {
						continue
					}
					key := keySelector(item)
					set[key] = append(set[key], elementSelector(item))
				}

				length := len(order)
				index := 0
				return func() (item KV[K, *[]V], ok bool) {
					ok = index < length
					if ok {
						key := order[index]
						slice := set[key]
						item = KV[K, *[]V]{key, &slice}
						index++
					}
					return
				}
			},
		}
	}

	return Query[KV[K, *[]V]]{
		iterate: func() func() (KV[K, *[]V], bool) {
			next := q.iterate()
			set := make(map[K][]V)
			var keys []K
			for item, ok := next(); ok; item, ok = next() {
				key := keySelector(item)
				if _, ok := set[key]; !ok {
					keys = append(keys, key)
				}
				set[key] = append(set[key], elementSelector(item))
			}
			length := len(keys)
			index := 0
			return func() (item KV[K, *[]V], ok bool) {
				ok = index < length
				if ok {
					key := keys[index]
					// 返回切片的指针以满足 comparable 约束
					slice := set[key]
					item = KV[K, *[]V]{key, &slice}
					index++
				}
				return
			}
		},
	}
}

// Select 将序列中的每个元素转换为新的形式
func Select[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	// 即使因为类型改变(T->V)无法传递 fastSlice，我们依然可以优化 Select 自身的迭代过程
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere // 可能为 nil

		iterator := func() func() (V, bool) {
			index := 0
			length := len(source)
			return func() (item V, ok bool) {
				for index < length {
					t := source[index]
					index++
					// 如果有上游 Where 条件，先检查
					if predicate != nil && !predicate(t) {
						continue
					}
					// 应用 Select
					return selector(t), true
				}
				return
			}
		}
		return Query[V]{
			iterate:  iterator,
			capacity: len(source),
		}
	}

	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			return func() (item V, ok bool) {
				var it T
				it, ok = next()
				if ok {
					item = selector(it)
				}
				return
			}
		},
		capacity: q.capacity,
	}
}

// SelectAsync 并发地转换序列中的每个元素
// 注意：结果的顺序不能保证与源序列一致
// 警告：如果不消费完所有结果，请使用 SelectAsyncCtx 以避免 goroutine 泄漏
func SelectAsync[T, V comparable](q Query[T], workers int, selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			// 使用足够大的 buffer 来减少阻塞风险
			// 但仍然存在泄漏风险，建议使用 SelectAsyncCtx
			outCh := make(chan V, workers*2)
			doneCh := make(chan struct{})
			var closeOnce sync.Once

			go func() {
				defer close(outCh)
				sem := make(chan struct{}, workers)
				var wg sync.WaitGroup

				if q.fastSlice != nil {
					source := q.fastSlice
					predicate := q.fastWhere

					for _, item := range source {
						if predicate != nil && !predicate(item) {
							continue
						}

						select {
						case <-doneCh:
							wg.Wait()
							return
						case sem <- struct{}{}:
							wg.Add(1)
							go func(it T) {
								defer wg.Done()
								defer func() {
									<-sem
									if r := recover(); r != nil {
										_ = r
									}
								}()
								result := selector(it)
								select {
								case <-doneCh:
									return
								case outCh <- result:
								}
							}(item)
						}
					}
					wg.Wait()
					return
				}

				// Slow Path: Iterator
				for item, ok := next(); ok; item, ok = next() {
					select {
					case <-doneCh:
						wg.Wait()
						return
					case sem <- struct{}{}:
						wg.Add(1)
						go func(it T) {
							defer wg.Done()
							defer func() {
								<-sem
								if r := recover(); r != nil {
									// 处理 selector panic
									_ = r
								}
							}()
							result := selector(it)
							select {
							case <-doneCh:
								return
							case outCh <- result:
							}
						}(item)
					}
				}
				wg.Wait()
			}()

			return func() (item V, ok bool) {
				item, ok = <-outCh
				if !ok {
					// 确保 worker goroutines 停止
					closeOnce.Do(func() {
						close(doneCh)
					})
				}
				return
			}
		},
		capacity: q.capacity,
	}
}

// SelectAsyncCtx 并发地转换序列中的每个元素，支持 Context 取消
// 当 ctx 被取消时，后台 goroutine 会安全退出，避免泄漏
func SelectAsyncCtx[T, V comparable](ctx context.Context, q Query[T], workers int, selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			outCh := make(chan V, workers*2)
			// 使用独立的 done channel 来协调关闭，避免竞争
			doneCh := make(chan struct{})
			var closeOnce sync.Once

			go func() {
				defer close(outCh)
				sem := make(chan struct{}, workers)
				var wg sync.WaitGroup

				if q.fastSlice != nil {
					source := q.fastSlice
					predicate := q.fastWhere

					for _, item := range source {
						// 检查 context
						select {
						case <-ctx.Done():
							closeOnce.Do(func() { close(doneCh) })
							wg.Wait()
							return
						case <-doneCh:
							wg.Wait()
							return
						default:
						}

						if predicate != nil && !predicate(item) {
							continue
						}

						select {
						case <-ctx.Done():
							closeOnce.Do(func() { close(doneCh) })
							wg.Wait()
							return
						case <-doneCh:
							wg.Wait()
							return
						case sem <- struct{}{}:
							wg.Add(1)
							go func(it T) {
								defer wg.Done()
								defer func() {
									<-sem
									if r := recover(); r != nil {
										_ = r
									}
								}()
								result := selector(it)
								select {
								case <-ctx.Done():
									return
								case <-doneCh:
									return
								case outCh <- result:
								}
							}(item)
						}
					}
					wg.Wait()
					return
				}

				// Slow Path: Iterator
				for item, ok := next(); ok; item, ok = next() {
					// 检查 context 是否已取消
					select {
					case <-ctx.Done():
						closeOnce.Do(func() { close(doneCh) })
						wg.Wait()
						return
					case <-doneCh:
						wg.Wait()
						return
					default:
					}

					select {
					case <-ctx.Done():
						closeOnce.Do(func() { close(doneCh) })
						wg.Wait()
						return
					case <-doneCh:
						wg.Wait()
						return
					case sem <- struct{}{}:
						wg.Add(1)
						go func(it T) {
							defer wg.Done()
							defer func() {
								<-sem
								if r := recover(); r != nil {
									_ = r
								}
							}()
							result := selector(it)
							select {
							case <-ctx.Done():
								return
							case <-doneCh:
								return
							case outCh <- result:
							}
						}(item)
					}
				}
				wg.Wait()
			}()

			var closed bool
			return func() (item V, ok bool) {
				if closed {
					return
				}
				select {
				case <-ctx.Done():
					closeOnce.Do(func() { close(doneCh) })
					closed = true
					return
				case item, ok = <-outCh:
					if !ok {
						closed = true
					}
					return
				}
			}
		},
	}
}

// Filter 根据选择器返回的布尔值过滤元素，并转换类型
func Filter[T, V comparable](q Query[T], selector func(T) (V, bool)) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					item, ok = selector(it)
					if ok {
						return
					}
				}
				return
			}
		},
	}
}

// Distinct 根据选择器返回的值对序列进行去重
// Distinct[T, V] 对于 T 类型的序列，使用 selector(T) -> V 进行去重，返回 V 类型的序列
func Distinct[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			set := make(map[V]struct{})
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; !has {
						set[s] = struct{}{}
						item = s
						return
					}
				}
				return
			}
		},
	}
}

// ExceptBy 根据选择器返回的值计算差集
// 返回在第一个序列中但不在第二个序列中的元素（基于选择器返回值）
func ExceptBy[T, V comparable](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[V]struct{})
			for i, ok := next2(); ok; i, ok = next2() {
				s := selector(i)
				set[s] = struct{}{}
			}
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; !has {
						item = s
						return
					}
				}
				return
			}
		},
	}
}

// Range 生成一个整数序列
func Range[T Integer](start, count T) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index T
			current := start
			return func() (item T, ok bool) {
				if index >= count {
					return
				}
				item, ok = current, true
				index++
				current++
				return
			}
		},
	}
}

// Repeat 生成包含同一个元素的序列
func Repeat[T cmp.Ordered](value T, count int) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index int
			return func() (item T, ok bool) {
				if index >= count {
					return
				}
				item, ok = value, true
				index++
				return
			}
		},
	}
}

// IntersectBy 根据选择器返回的值计算交集
func IntersectBy[T, V comparable](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[V]struct{})
			for item, ok := next2(); ok; item, ok = next2() {
				s := selector(item)
				set[s] = struct{}{}
			}
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; has {
						delete(set, s)
						item = s
						return
					}
				}
				return
			}
		},
	}
}

// ToMap 将序列转换为 map，需要提供 Key 选择器
func ToMap[T, K comparable](q Query[T], selector func(T) K) map[K]T {
	ret := make(map[K]T)
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		k := selector(item)
		ret[k] = item
	}
	return ret
}

// Uniq 返回去重后的切片
func Uniq[T comparable](list []T) []T {
	result := []T{}
	seen := map[T]struct{}{}
	for _, e := range list {
		if _, ok := seen[e]; ok {
			continue
		}
		result = append(result, e)
		seen[e] = struct{}{}
	}
	return result
}

// Contains 判断切片是否包含指定元素
func Contains[T comparable](list []T, element T) bool {
	for _, item := range list {
		if item == element {
			return true
		}
	}
	return false
}

// IndexOf 返回元素在切片中的索引，未找到返回 -1
func IndexOf[T comparable](list []T, element T) int {
	for i, item := range list {
		if item == element {
			return i
		}
	}
	return -1
}

// LastIndexOf 返回元素在切片中最后一次出现的索引，未找到返回 -1
func LastIndexOf[T comparable](list []T, element T) int {
	length := len(list)
	for i := length - 1; i >= 0; i-- {
		if list[i] == element {
			return i
		}
	}
	return -1
}

// Reverse 反转切片中的元素，返回新切片，原切片不变
func Reverse[T comparable](list []T) []T {
	result := make([]T, len(list))
	copy(result, list)
	length := len(result)
	half := length / 2
	for i := 0; i < half; i++ {
		j := length - 1 - i
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Min 返回切片中的最小值
func Min[T cmp.Ordered](list ...T) T {
	var min T
	if len(list) == 0 {
		return min
	}
	min = list[0]
	for i := 1; i < len(list); i++ {
		item := list[i]
		if item < min {
			min = item
		}
	}
	return min
}

// Max 返回切片中的最大值
func Max[T cmp.Ordered](list ...T) T {
	var max T
	if len(list) == 0 {
		return max
	}
	max = list[0]
	for i := 1; i < len(list); i++ {
		item := list[i]
		if item > max {
			max = item
		}
	}
	return max
}

// MinBy 根据选择器返回的值计算最小值
func MinBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	first := true
	for item, ok := next(); ok; item, ok = next() {
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

// MaxBy 根据选择器返回的值计算最大值
func MaxBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	first := true
	for item, ok := next(); ok; item, ok = next() {
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

// SumBy 根据选择器返回的值计算总和
func SumBy[T comparable, V Integer | Float | Complex](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}

// AvgBy 计算平均值，兼容所有类型
func AvgBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) float64 {
	next := q.iterate()
	var sum float64
	var n int
	for item, ok := next(); ok; item, ok = next() {
		sum += float64(selector(item))
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// Sum 计算切片中所有元素的总和
func Sum[T Float | Integer | Complex](list []T) T {
	var sum T = 0
	for _, val := range list {
		sum += val
	}
	return sum
}

// Every 判断 list 中的所有元素是否都存在于 subset 中
func Every[T comparable](list []T, subset []T) bool {
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; !ok {
			return false
		}
	}
	return true
}

// Some 判断 subset 中是否至少有一个元素存在于 list 中
func Some[T comparable](list []T, subset []T) bool {
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; ok {
			return true
		}
	}
	return false
}

// None 判断 subset 中的所有元素是否都不存在于 list 中
func None[T comparable](list []T, subset []T) bool {
	seen := make(map[T]struct{}, len(list))
	for _, elem := range list {
		seen[elem] = struct{}{}
	}
	for _, elem := range subset {
		if _, ok := seen[elem]; ok {
			return false
		}
	}
	return true
}

// Intersect 返回两个切片的交集
func Intersect[T comparable](list1 []T, list2 []T) []T {
	result := []T{}
	seen := map[T]struct{}{}
	for _, elem := range list1 {
		seen[elem] = struct{}{}
	}
	for _, elem := range list2 {
		if _, ok := seen[elem]; ok {
			result = append(result, elem)
		}
	}
	return result
}

// Difference 计算两个切片的差异，返回 (list1-list2, list2-list1)
func Difference[T comparable](list1 []T, list2 []T) ([]T, []T) {
	left := []T{}
	right := []T{}
	seenLeft := map[T]struct{}{}
	seenRight := map[T]struct{}{}
	for _, elem := range list1 {
		seenLeft[elem] = struct{}{}
	}
	for _, elem := range list2 {
		seenRight[elem] = struct{}{}
	}
	for _, elem := range list1 {
		if _, ok := seenRight[elem]; !ok {
			left = append(left, elem)
		}
	}
	for _, elem := range list2 {
		if _, ok := seenLeft[elem]; !ok {
			right = append(right, elem)
		}
	}
	return left, right
}

// Union 返回两个切片的并集，自动去重
func Union[T comparable](list1 []T, list2 []T) []T {
	result := make([]T, 0, len(list1)+len(list2))
	seen := make(map[T]struct{})
	for _, e := range list1 {
		if _, ok := seen[e]; !ok {
			seen[e] = struct{}{}
			result = append(result, e)
		}
	}
	for _, e := range list2 {
		if _, ok := seen[e]; !ok {
			seen[e] = struct{}{}
			result = append(result, e)
		}
	}
	return result
}

// Without 从切片中移除指定的元素
func Without[T comparable](list []T, exclude ...T) []T {
	excludeSet := make(map[T]struct{}, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = struct{}{}
	}
	result := make([]T, 0, len(list))
	for _, e := range list {
		if _, ok := excludeSet[e]; !ok {
			result = append(result, e)
		}
	}
	return result
}

// NoEmpty 移除切片中的空值（零值）
func NoEmpty[T comparable](list []T) []T {
	var empty T
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e != empty {
			result = append(result, e)
		}
	}
	return result
}

// GtZero 移除切片中不大于 0 的值
func GtZero[T Float | Integer](list []T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e > 0 {
			result = append(result, e)
		}
	}
	return result
}

// Rand 随机从切片中选取 count 个元素
func Rand[T comparable](list []T, count int) []T {
	size := len(list)
	if count > size {
		count = size
	}
	if count <= 0 {
		return []T{}
	}
	// 预分配临时切片和结果切片
	templist := make([]T, size)
	copy(templist, list)
	results := make([]T, 0, count)

	// Fisher-Yates shuffle variant (partial shuffle)
	for i := 0; i < count; i++ {
		// Pick random index from remaining elements
		remaining := size - i
		index := rand.IntN(remaining)

		// Add picked element
		results = append(results, templist[index])

		// Move last unpicked element to picked position (swap-remove)
		// We don't actually need to swap if we just want to fill 'results',
		// simply overwriting the picked slot with the last valid element is enough
		// because we shorten the range in next iteration.
		templist[index] = templist[remaining-1]
	}
	return results
}

// Shuffle 随机打乱切片中的元素，返回新切片，原切片不变
func Shuffle[T comparable](list []T) []T {
	result := make([]T, len(list))
	copy(result, list)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Default 如果值为空（零值），返回默认值
func Default[T comparable](v, d T) T {
	if IsEmpty(v) {
		return d
	}
	return v
}

// Empty 返回类型的零值
func Empty[T comparable]() T {
	var zero T
	return zero
}

// IsEmpty 判断值是否为空（零值）
func IsEmpty[T comparable](v T) bool {
	var zero T
	return zero == v
}

// IsNotEmpty 判断值是否不为空（非零值）
func IsNotEmpty[T comparable](v T) bool {
	var zero T
	return zero != v
}
func try(callback func() error) (ok bool) {
	ok = true
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	err := callback()
	if err != nil {
		ok = false
	}
	return
}

// Try 尝试执行函数，支持重试和延迟
func Try(callback func() error, nums ...int) bool {
	num, second := 1, 0
	if len(nums) > 0 {
		num = nums[0]
	}
	if len(nums) > 1 {
		second = nums[1]
	}
	var i int
	for i < num {
		if try(callback) {
			return true
		}
		if second > 0 {
			time.Sleep(time.Duration(second) * time.Second)
		}
		i++
	}
	return false
}

// TryCatch 尝试执行函数，如果 panic 则执行 catch 函数
func TryCatch(callback func() error, catch func()) {
	if !try(callback) {
		catch()
	}
}

// IF 三目运算
func IF[T comparable](cond bool, suc, fail T) T {
	if cond {
		return suc
	} else {
		return fail
	}
}
