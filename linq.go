package linq

import (
	crand "crypto/rand"
	"math/big"
	"math/rand/v2"
	"sync"
	"time"
	"unicode/utf8"
)

type lesserFunc[T any] func([]T) func(i, j int) bool

// KV 键值对结构体，用于存储分组等操作的结果
type KV[K comparable, V any] struct {
	Key   K
	Value V
}

// Query 查询结构体，是 LINQ 操作的核心类型
type Query[T any] struct {
	lesser  lesserFunc[T]
	iterate func() func() (T, bool)
}

// From 从切片创建 Query 查询对象
func From[T any](source []T) Query[T] {
	len := len(source)
	return Query[T]{
		iterate: func() func() (T, bool) {
			index := 0
			return func() (item T, ok bool) {
				ok = index < len
				if ok {
					item = source[index]
					index++
				}
				return
			}
		},
	}
}

// FromChannel 从只读 Channel 创建 Query 查询对象
func FromChannel[T any](source <-chan T) Query[T] {
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
func FromMap[K comparable, V any](source map[K]V) Query[KV[K, V]] {
	len := len(source)
	keyvalues := make([](KV[K, V]), 0, len)
	for key, value := range source {
		keyvalues = append(keyvalues, KV[K, V]{Key: key, Value: value})
	}
	return From(keyvalues)
}

// Where 返回满足指定条件的元素序列
func (q Query[T]) Where(predicate func(T) bool) Query[T] {
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
	}
}

// Skip 跳过前 N 个元素
func (q Query[T]) Skip(count int) Query[T] {
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

// Page 分页查询，返回指定页码和页大小的元素
func (q Query[T]) Page(page, pageSize int) Query[T] {
	return q.Skip((page - 1) * pageSize).Take(pageSize)
}

// Union 返回两个序列的并集，自动去重
func (q Query[T]) Union(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
			set := make(map[any]struct{})
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
			set := make(map[any]struct{})
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
			set := make(map[any]struct{})
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
	_, ok := q.iterate()()
	return ok
}

// AnyWith 判断是否存在满足条件的元素
func (q Query[T]) AnyWith(predicate func(T) bool) bool {
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
	item, _ := q.iterate()()
	return item
}

// FirstWith 返回第一个满足条件的元素
func (q Query[T]) FirstWith(predicate func(T) bool) T {
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
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !action(item) {
			return
		}
	}
}

// ForEachIndexed 带索引遍历序列中的每个元素
func (q Query[T]) ForEachIndexed(action func(int, T) bool) {
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

// Last 返回序列的最后一个元素
func (q Query[T]) Last() (r T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = item
	}
	return
}

// LastWith 返回最后一个满足条件的元素
func (q Query[T]) LastWith(predicate func(T) bool) (r T) {
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
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			var items []T
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
func (q Query[T]) SumInt8By(selector func(T) int8) (r int8) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumInt16By(selector func(T) int16) (r int16) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumIntBy(selector func(T) int) (r int) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumInt32By(selector func(T) int32) (r int32) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumInt64By(selector func(T) int64) (r int64) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumUInt8By(selector func(T) uint8) (r uint8) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumUInt16By(selector func(T) uint16) (r uint16) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumUIntBy(selector func(T) uint) (r uint) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumUInt32By(selector func(T) uint32) (r uint32) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumUInt64By(selector func(T) uint64) (r uint64) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumFloat32By(selector func(T) float32) (r float32) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumFloat64By(selector func(T) float64) (r float64) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}

func (q Query[T]) AvgIntBy(selector func(T) int) float64 {
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
func (q Query[T]) AvgInt64By(selector func(T) int64) float64 {
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
func (q Query[T]) AvgBy(selector func(T) float64) float64 {
	next := q.iterate()
	var sum float64
	var n int
	for item, ok := next(); ok; item, ok = next() {
		sum += selector(item)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func (q Query[T]) Count() (r int) {
	next := q.iterate()
	for _, ok := next(); ok; _, ok = next() {
		r++
	}
	return
}
func (q Query[T]) ToSlice() (r []T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, item)
	}
	return
}
func (q Query[T]) AppendTo(dest []T) []T {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		dest = append(dest, item)
	}
	return dest
}
func (q Query[T]) ToChannel(c chan<- T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		c <- item
	}
	close(c)
}
func (q Query[T]) ToMap(selector func(T) map[string]any) (r []map[string]any) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, selector(item))
	}
	return
}

func GroupBy[T any, K comparable](q Query[T], keySelector func(T) K) Query[KV[K, []T]] {
	return Query[KV[K, []T]]{
		iterate: func() func() (KV[K, []T], bool) {
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
			len := len(keys)
			index := 0
			return func() (item KV[K, []T], ok bool) {
				ok = index < len
				if ok {
					key := keys[index]
					item = KV[K, []T]{key, set[key]}
					index++
				}
				return
			}
		},
	}
}
func GroupBySelect[T any, K comparable, V any](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[KV[K, []V]] {
	return Query[KV[K, []V]]{
		iterate: func() func() (KV[K, []V], bool) {
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
			len := len(keys)
			index := 0
			return func() (item KV[K, []V], ok bool) {
				ok = index < len
				if ok {
					key := keys[index]
					item = KV[K, []V]{key, set[key]}
					index++
				}
				return
			}
		},
	}
}
func Select[T, V any](q Query[T], selector func(T) V) Query[V] {
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
	}
}

// SelectAsync performs the selector function concurrently.
// Note: The order of elements in the result is NOT guaranteed to match the source.
// WARNING: If you don't consume all results, use SelectAsyncCtx to avoid goroutine leaks.
func SelectAsync[T, V any](q Query[T], workers int, selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			// 使用足够大的 buffer 来减少阻塞风险
			// 但仍然存在泄漏风险，建议使用 SelectAsyncCtx
			outCh := make(chan V, workers*2)
			doneCh := make(chan struct{})

			go func() {
				defer close(outCh)
				sem := make(chan struct{}, workers)
				var wg sync.WaitGroup

				for item, ok := next(); ok; item, ok = next() {
					select {
					case <-doneCh:
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

			var closed bool
			return func() (item V, ok bool) {
				if closed {
					return
				}
				item, ok = <-outCh
				if !ok {
					closed = true
					close(doneCh)
				}
				return
			}
		},
	}
}
func Filter[T, V any](q Query[T], selector func(T) (V, bool)) Query[V] {
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
func Distinct[T, V any](q Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			set := make(map[any]struct{})
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; !has {
						set[s] = struct{}{}
						return
					}
				}
				return
			}
		},
	}
}
func ExceptBy[T, V any](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
func Range[T Integer](start, count T) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index T
			current := start
			var it T
			return func() (item T, ok bool) {
				if index >= count {
					return it, false
				}
				item, ok = current, true
				index++
				current++
				return
			}
		},
	}
}
func Repeat[T Ordered](value T, count int) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index int
			var it T
			return func() (item T, ok bool) {
				if index >= count {
					return it, false
				}
				item, ok = value, true
				index++
				return
			}
		},
	}
}
func IntersectBy[T, V any](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
func ToMap[T, K comparable](q Query[T], selector func(T) K) map[K]T {
	ret := make(map[K]T)
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		k := selector(item)
		ret[k] = item
	}
	return ret
}

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
func Contains[T comparable](list []T, element T) bool {
	for _, item := range list {
		if item == element {
			return true
		}
	}
	return false
}
func IndexOf[T comparable](list []T, element T) int {
	for i, item := range list {
		if item == element {
			return i
		}
	}
	return -1
}
func LastIndexOf[T comparable](list []T, element T) int {
	length := len(list)
	for i := length - 1; i >= 0; i-- {
		if list[i] == element {
			return i
		}
	}
	return -1
}
func Shuffle[T any](list []T) []T {
	result := make([]T, len(list))
	copy(result, list)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}
func Reverse[T any](list []T) []T {
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
func Min[T Ordered](list ...T) T {
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
func Max[T Ordered](list ...T) T {
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
func MinBy[T any, V Integer | Float](q Query[T], selector func(T) V) (r V) {
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
func MaxBy[T any, V Integer | Float](q Query[T], selector func(T) V) (r V) {
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
func SumBy[T any, V Integer | Float | Complex](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func AvgBy[T any](q Query[T], selector func(T) float64) float64 {
	next := q.iterate()
	var sum float64
	var n int
	for item, ok := next(); ok; item, ok = next() {
		sum += selector(item)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func Sum[T Float | Integer | Complex](list []T) T {
	var sum T = 0
	for _, val := range list {
		sum += val
	}
	return sum
}

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
func GtZero[T Float | Integer](list []T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e > 0 {
			result = append(result, e)
		}
	}
	return result
}
func cryptoRandIntn(n int) int {
	max := big.NewInt(int64(n))
	i, err := crand.Int(crand.Reader, max)
	if err != nil {
		return 0
	}
	return int(i.Int64())
}
func Rand[T any](list []T, count int) []T {
	size := len(list)
	templist := append([]T{}, list...)
	results := []T{}
	for i := 0; i < size && i < count; i++ {
		copyLength := size - i
		index := cryptoRandIntn(size - i)
		results = append(results, templist[index])
		templist[index] = templist[copyLength-1]
		templist = templist[:copyLength-1]
	}
	return results
}
func Default[T comparable](v, d T) T {
	if IsEmpty(v) {
		return d
	}
	return v
}
func Empty[T any]() T {
	var zero T
	return zero
}
func IsEmpty[T comparable](v T) bool {
	var zero T
	return zero == v
}
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
func TryCatch(callback func() error, catch func()) {
	if !try(callback) {
		catch()
	}
}

// 三目运算int
func IF[T any](cond bool, suc, fail T) T {
	if cond {
		return suc
	} else {
		return fail
	}
}
