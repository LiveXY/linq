package linq

import (
	"cmp"
	"context"
	"math/rand/v2"
	"slices"
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
	lesser    lesserFunc[T]
	iterate   func() func() (T, bool)
	fastSlice []T
	fastWhere func(T) bool
	capacity  int
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
		capacity: len(source),
	}
}

// FromMap 从 Map 创建 Query 查询对象，每个元素为 KV 键值对
func FromMap[K, V comparable](source map[K]V) Query[KV[K, V]] {
	length := len(source)
	if length == 0 {
		return From([]KV[K, V]{})
	}

	return Query[KV[K, V]]{
		iterate: func() func() (KV[K, V], bool) {
			// 如果调用了多次 iterate()，我们还是在调用时决定是否快照
			// 或者直接使用 map 迭代器。为了保证并发迭代下的稳定性，第一次调用时转为 slice 是可接受的
			// 但我们将这个步骤延迟到这个闭包被执行时
			keyvalues := make([](KV[K, V]), 0, length)
			for key, value := range source {
				keyvalues = append(keyvalues, KV[K, V]{Key: key, Value: value})
			}

			index := 0
			return func() (item KV[K, V], ok bool) {
				if index < length {
					item = keyvalues[index]
					index++
					return item, true
				}
				return
			}
		},
		capacity: length,
	}
}

// Where 返回满足指定条件的查询对象
func (q Query[T]) Where(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
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
			capacity:  q.capacity,
		}
	}
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if predicate(item) {
						return item, true
					}
				}
				return
			}
		},
		capacity: q.capacity,
	}
}

// Skip 跳过前 N 个的查询对象
func (q Query[T]) Skip(count int) Query[T] {
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

// Take 获取前 N 个的查询对象
func (q Query[T]) Take(count int) Query[T] {
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

// TakeWhile 获取满足条件的元素，直到遇到不满足条件的查询对象
func (q Query[T]) TakeWhile(predicate func(T) bool) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
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

// SkipWhile 跳过满足条件的元素，直到遇到不满足条件的的查询对象
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

// Page 分页查询，返回指定页码和页大小的的查询对象
func (q Query[T]) Page(page, pageSize int) Query[T] {
	return q.Skip((page - 1) * pageSize).Take(pageSize)
}

// Append 在序列末尾追加一个元素的查询对象
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

// Concat 连接两个序列的查询对象
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

// Prepend 在序列开头插入一个元素的查询对象
func (q Query[T]) Prepend(item T) Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		iterator := func() func() (T, bool) {
			index := 0
			length := len(source)
			prepended := false
			return func() (T, bool) {
				if !prepended {
					prepended = true
					return item, true
				}
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

// DefaultIfEmpty 如果序列为空，返回包含默认值的序列的查询对象
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

// Distinct 返回去重后的序列的查询对象
func (q Query[T]) Distinct() Query[T] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[T]{
			iterate: func() func() (T, bool) {
				index := 0
				length := len(source)
				set := make(map[T]struct{}, length)
				return func() (item T, ok bool) {
					for index < length {
						it := source[index]
						index++
						if predicate != nil && !predicate(it) {
							continue
						}
						if _, has := set[it]; !has {
							set[it] = struct{}{}
							return it, true
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
			capacity := q.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[T]struct{}, capacity)
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

// Intersect 返回交集，即同时存在于两个序列中的元素的查询对象
func (q Query[T]) Intersect(q2 Query[T]) Query[T] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		iterator := func() func() (T, bool) {
			index := 0
			length := len(s1)
			set := make(map[T]struct{}, len(s2))
			for _, item := range s2 {
				if p2 != nil && !p2(item) {
					continue
				}
				set[item] = struct{}{}
			}
			return func() (item T, ok bool) {
				for index < length {
					item = s1[index]
					index++
					if p1 != nil && !p1(item) {
						continue
					}
					if _, has := set[item]; has {
						delete(set, item)
						return item, true
					}
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
			capacity := q2.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[T]struct{}, capacity)
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

// Union 返回两个序列的并集，自动去重的查询对象
func (q Query[T]) Union(q2 Query[T]) Query[T] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		return Query[T]{
			iterate: func() func() (T, bool) {
				idx1 := 0
				idx2 := 0
				len1 := len(s1)
				len2 := len(s2)
				set := make(map[T]struct{}, len1+len2)
				return func() (item T, ok bool) {
					for idx1 < len1 {
						item = s1[idx1]
						idx1++
						if p1 != nil && !p1(item) {
							continue
						}
						if _, has := set[item]; !has {
							set[item] = struct{}{}
							return item, true
						}
					}
					for idx2 < len2 {
						item = s2[idx2]
						idx2++
						if p2 != nil && !p2(item) {
							continue
						}
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
			capacity := q.capacity + q2.capacity
			if capacity == 0 {
				capacity = 16
			}
			set := make(map[T]struct{}, capacity)
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

// Except 返回差集，即在第一个序列中但不在第二个序列中的元素的查询对象
func (q Query[T]) Except(q2 Query[T]) Query[T] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		iterator := func() func() (T, bool) {
			index := 0
			length := len(s1)
			set := make(map[T]struct{}, len(s2))
			for _, item := range s2 {
				if p2 != nil && !p2(item) {
					continue
				}
				set[item] = struct{}{}
			}
			return func() (item T, ok bool) {
				for index < length {
					item = s1[index]
					index++
					if p1 != nil && !p1(item) {
						continue
					}
					if _, has := set[item]; !has {
						return item, true
					}
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
			capacity := q2.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[T]struct{}, capacity)
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

// Reverse 返回反转后的序列的查询对象
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
		return slices.ContainsFunc(q.fastSlice, q.fastWhere)
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

// FirstIfEmpty 返回序列的第一个元素 如果为空返回默认值
func (q Query[T]) FirstDefault(d ...T) T {
	var v = q.First()
	if len(d) == 0 {
		return Empty[T]()
	}
	if IsEmpty(v) {
		return d[0]
	}
	return v
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
func (q Query[T]) ForEach(predicate func(T) bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if !predicate(item) {
				return
			}
		}
		return
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !predicate(item) {
			return
		}
	}
}

// ForEachIndexed 带索引遍历序列中的每个元素
func (q Query[T]) ForEachIndexed(predicate func(int, T) bool) {
	if q.fastSlice != nil {
		source := q.fastSlice
		preFilter := q.fastWhere
		index := 0
		for _, item := range source {
			if preFilter != nil && !preFilter(item) {
				continue
			}
			if !predicate(index, item) {
				return
			}
			index++
		}
		return
	}
	next := q.iterate()
	index := 0
	for item, ok := next(); ok; item, ok = next() {
		if !predicate(index, item) {
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

	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		length := len(source)
		if length == 0 {
			return
		}

		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(wIdx int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						_ = r
					}
				}()
				// Stride based distribution implies simple load balancing
				for j := wIdx; j < length; j += workers {
					item := source[j]
					if predicate != nil && !predicate(item) {
						continue
					}
					action(item)
				}
			}(i)
		}
		wg.Wait()
		return
	}

	ch := make(chan T, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
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

	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		length := len(source)
		if length == 0 {
			return
		}

		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(wIdx int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						_ = r
					}
				}()
				for j := wIdx; j < length; j += workers {
					select {
					case <-ctx.Done():
						return
					default:
					}
					item := source[j]
					if predicate != nil && !predicate(item) {
						continue
					}
					action(item)
				}
			}(i)
		}
		wg.Wait()
		return
	}

	ch := make(chan T, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
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

// LastIfEmpty 返返回序列的最后一个元素 如果为空返回默认值
func (q Query[T]) LastDefault(d ...T) T {
	var v = q.Last()
	if len(d) == 0 {
		return Empty[T]()
	}
	if IsEmpty(v) {
		return d[0]
	}
	return v
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

// Single 返回序列中的唯一元素，如果序列为空或包含多个元素则返回零值
func (q Query[T]) Single() (r T) {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		found := false
		if predicate == nil {
			if len(source) == 1 {
				return source[0]
			}
			return
		}
		for _, item := range source {
			if predicate(item) {
				if found {
					var zero T
					return zero
				}
				r = item
				found = true
			}
		}
		if found {
			return r
		}
		var zero T
		return zero
	}
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
	if q.fastSlice != nil && q.fastWhere == nil {
		return len(q.fastSlice)
	}
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
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		if predicate == nil {
			r = make([]T, len(source))
			copy(r, source)
			return
		}
		r = make([]T, 0, q.capacity)
		for _, item := range source {
			if predicate(item) {
				r = append(r, item)
			}
		}
		return
	}
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
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		if predicate == nil {
			dest = slices.Grow(dest, len(source))
			dest = append(dest, source...)
			return dest
		}
		for _, item := range source {
			if predicate(item) {
				dest = append(dest, item)
			}
		}
		return dest
	}
	if q.capacity > 0 {
		dest = slices.Grow(dest, q.capacity)
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		dest = append(dest, item)
	}
	return dest
}

// ToChannel 将序列写入到指定的只写 Channel
func (q Query[T]) ToChannel(c chan<- T) {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		for _, item := range source {
			if predicate != nil && !predicate(item) {
				continue
			}
			c <- item
		}
		close(c)
		return
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		c <- item
	}
	close(c)
}

// ToMapSlice 将序列转换为 []map[string]T，通常用于 JSON 序列化
func (q Query[T]) ToMapSlice(selector func(T) map[string]T) (r []map[string]T) {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		if predicate == nil {
			r = make([]map[string]T, len(source))
			for i, item := range source {
				r[i] = selector(item)
			}
			return
		}
		r = make([]map[string]T, 0, q.capacity)
		for _, item := range source {
			if predicate(item) {
				r = append(r, selector(item))
			}
		}
		return
	}
	if q.capacity > 0 {
		r = make([]map[string]T, 0, q.capacity)
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, selector(item))
	}
	return
}

// GroupBy 根据 keySelector 对序列进行分组的查询对象
func GroupBy[T, K comparable](q Query[T], keySelector func(T) K) Query[KV[K, *[]T]] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[KV[K, *[]T]]{
			iterate: func() func() (KV[K, *[]T], bool) {
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
				set := make(map[K][]T, len(counts))
				for _, key := range order {
					set[key] = make([]T, 0, counts[key])
				}
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
					slice := set[key]
					item = KV[K, *[]T]{key, &slice}
					index++
				}
				return
			}
		},
	}
}

// GroupBySelect 根据 keySelector 对序列进行分组附带选择器的查询对象
func GroupBySelect[T, K, V comparable](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[KV[K, *[]V]] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[KV[K, *[]V]]{
			iterate: func() func() (KV[K, *[]V], bool) {
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
				set := make(map[K][]V, len(counts))
				for _, key := range order {
					set[key] = make([]V, 0, counts[key])
				}
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
					slice := set[key]
					item = KV[K, *[]V]{key, &slice}
					index++
				}
				return
			}
		},
	}
}

// Select 将序列中的每个元素转换为新的对象的查询对象
func Select[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				index := 0
				length := len(source)
				return func() (item V, ok bool) {
					for index < length {
						t := source[index]
						index++
						if predicate != nil && !predicate(t) {
							continue
						}
						return selector(t), true
					}
					return
				}
			},
			capacity: q.capacity, // 传递容量信息
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
			outCh := make(chan V, workers*2)
			doneCh := make(chan struct{})
			var closeOnce sync.Once

			go func() {
				defer close(outCh)

				if q.fastSlice != nil {
					source := q.fastSlice
					predicate := q.fastWhere
					length := len(source)
					var wg sync.WaitGroup
					wg.Add(workers)

					for i := 0; i < workers; i++ {
						go func(wIdx int) {
							defer wg.Done()
							defer func() {
								if r := recover(); r != nil {
									_ = r
								}
							}()
							for j := wIdx; j < length; j += workers {
								item := source[j]
								if predicate != nil && !predicate(item) {
									continue
								}
								result := selector(item)
								select {
								case <-doneCh:
									return
								case outCh <- result:
								}
							}
						}(i)
					}
					wg.Wait()
					return
				}

				next := q.iterate()
				sem := make(chan struct{}, workers)
				var wg sync.WaitGroup

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
			outCh := make(chan V, workers*2)
			doneCh := make(chan struct{})
			var closeOnce sync.Once

			go func() {
				defer close(outCh)

				if q.fastSlice != nil {
					source := q.fastSlice
					predicate := q.fastWhere
					length := len(source)
					var wg sync.WaitGroup
					wg.Add(workers)

					for i := 0; i < workers; i++ {
						go func(wIdx int) {
							defer wg.Done()
							defer func() {
								if r := recover(); r != nil {
									_ = r
								}
							}()
							for j := wIdx; j < length; j += workers {
								select {
								case <-ctx.Done():
									return
								case <-doneCh:
									return
								default:
								}
								item := source[j]
								if predicate != nil && !predicate(item) {
									continue
								}
								result := selector(item)
								select {
								case <-ctx.Done():
									return
								case <-doneCh:
									return
								case outCh <- result:
								}
							}
						}(i)
					}
					wg.Wait()
					return
				}

				next := q.iterate()
				sem := make(chan struct{}, workers)
				var wg sync.WaitGroup

				for item, ok := next(); ok; item, ok = next() {
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
		capacity: q.capacity,
	}
}

// WhereSelect 根据选择器返回的布尔值过滤元素，并转换类型的查询对象
func WhereSelect[T, V comparable](q Query[T], selector func(T) (V, bool)) Query[V] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				index := 0
				length := len(source)
				return func() (item V, ok bool) {
					for index < length {
						it := source[index]
						index++
						if predicate != nil && !predicate(it) {
							continue
						}
						item, ok = selector(it)
						if ok {
							return item, true
						}
					}
					return
				}
			},
			capacity: q.capacity,
		}
	}
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

// DistinctSelect 根据选择器返回的值对序列进行去重的查询对象
// DistinctSelect[T, V] 对于 T 类型的序列，使用 selector(T) -> V 进行去重，返回 V 类型的序列
func DistinctSelect[T, V comparable](q Query[T], selector func(T) V) Query[V] {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				index := 0
				length := len(source)
				set := make(map[V]struct{}, length)
				return func() (item V, ok bool) {
					for index < length {
						it := source[index]
						index++
						if predicate != nil && !predicate(it) {
							continue
						}
						s := selector(it)
						if _, has := set[s]; !has {
							set[s] = struct{}{}
							return s, true
						}
					}
					return
				}
			},
		}
	}
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			capacity := q.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[V]struct{}, capacity)
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

// UnionSelect 返回两个序列经过选择器处理后的并集，自动去重
func UnionSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				idx1 := 0
				idx2 := 0
				len1 := len(s1)
				len2 := len(s2)
				set := make(map[V]struct{}, len1+len2)
				return func() (item V, ok bool) {
					for idx1 < len1 {
						it := s1[idx1]
						idx1++
						if p1 != nil && !p1(it) {
							continue
						}
						val := selector(it)
						if _, has := set[val]; !has {
							set[val] = struct{}{}
							return val, true
						}
					}
					for idx2 < len2 {
						it := s2[idx2]
						idx2++
						if p2 != nil && !p2(it) {
							continue
						}
						val := selector(it)
						if _, has := set[val]; !has {
							set[val] = struct{}{}
							return val, true
						}
					}
					return
				}
			},
		}
	}
	return Query[V]{
		iterate: func() func() (V, bool) {
			next1 := q.iterate()
			next2 := q2.iterate()
			capacity := q.capacity + q2.capacity
			if capacity == 0 {
				capacity = 16
			}
			seen := make(map[V]struct{}, capacity)
			firstDone := false
			return func() (item V, ok bool) {
				if !firstDone {
					for it, ok1 := next1(); ok1; it, ok1 = next1() {
						val := selector(it)
						if _, has := seen[val]; !has {
							seen[val] = struct{}{}
							return val, true
						}
					}
					firstDone = true
				}
				for it, ok2 := next2(); ok2; it, ok2 = next2() {
					val := selector(it)
					if _, has := seen[val]; !has {
						seen[val] = struct{}{}
						return val, true
					}
				}
				return item, false
			}
		},
	}
}

// IntersectSelect 根据选择器返回的值计算交集的查询对象
func IntersectSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				index := 0
				length := len(s1)
				set := make(map[V]struct{}, len(s2))
				for _, item := range s2 {
					if p2 != nil && !p2(item) {
						continue
					}
					s := selector(item)
					set[s] = struct{}{}
				}
				return func() (item V, ok bool) {
					for index < length {
						it := s1[index]
						index++
						if p1 != nil && !p1(it) {
							continue
						}
						s := selector(it)
						if _, has := set[s]; has {
							delete(set, s)
							return s, true
						}
					}
					return
				}
			},
		}
	}
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			capacity := q2.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[V]struct{}, capacity)
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

// ExceptSelect 根据选择器返回的值计算差集的查询对象
// 返回在第一个序列中但不在第二个序列中的元素（基于选择器返回值）
func ExceptSelect[T, V comparable](q, q2 Query[T], selector func(T) V) Query[V] {
	if q.fastSlice != nil && q2.fastSlice != nil {
		s1 := q.fastSlice
		p1 := q.fastWhere
		s2 := q2.fastSlice
		p2 := q2.fastWhere
		return Query[V]{
			iterate: func() func() (V, bool) {
				index := 0
				length := len(s1)
				set := make(map[V]struct{}, len(s2))
				for _, item := range s2 {
					if p2 != nil && !p2(item) {
						continue
					}
					s := selector(item)
					set[s] = struct{}{}
				}
				return func() (item V, ok bool) {
					for index < length {
						it := s1[index]
						index++
						if p1 != nil && !p1(it) {
							continue
						}
						s := selector(it)
						if _, has := set[s]; !has {
							return s, true
						}
					}
					return
				}
			},
		}
	}
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			capacity := q2.capacity
			if capacity == 0 {
				capacity = 8
			}
			set := make(map[V]struct{}, capacity)
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

// Range 生成一个整数序列的查询对象
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
		capacity: int(count),
	}
}

// Repeat 生成包含同一个元素的序列的查询对象
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
		capacity: count,
	}
}

// ToMap 将序列转换为 map，需要提供 Key 选择器的查询对象
func ToMap[T, K comparable](q Query[T], selector func(T) K) map[K]T {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		var ret map[K]T
		if predicate == nil {
			ret = make(map[K]T, len(source))
			for _, item := range source {
				ret[selector(item)] = item
			}
			return ret
		}
		if q.capacity > 0 {
			ret = make(map[K]T, q.capacity)
		} else {
			ret = make(map[K]T)
		}
		for _, item := range source {
			if predicate(item) {
				ret[selector(item)] = item
			}
		}
		return ret
	}

	var ret map[K]T
	if q.capacity > 0 {
		ret = make(map[K]T, q.capacity)
	} else {
		ret = make(map[K]T)
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		k := selector(item)
		ret[k] = item
	}
	return ret
}

// Map 将序列中的每个元素转换为新的对象
func Map[T, V comparable](list []T, selector func(T) V) []V {
	return MapIndexed(list, func(item T, _ int) V { return selector(item) })
}

// MapIndexed 将序列中的每个元素转换为新的对象
func MapIndexed[T, V comparable](list []T, selector func(T, int) V) []V {
	result := make([]V, len(list))
	for i := range list {
		result[i] = selector(list[i], i)
	}
	return result
}

// Where 返回满足指定条件的元素序列
func Where[T comparable](list []T, predicate func(item T) bool) []T {
	return WhereIndexed(list, func(item T, _ int) bool { return predicate(item) })
}

// Where 返回满足指定条件的元素序列
func WhereIndexed[T comparable](list []T, predicate func(T, int) bool) []T {
	result := make([]T, 0, len(list))
	for i := range list {
		if predicate(list[i], i) {
			result = append(result, list[i])
		}
	}
	return result
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
	return slices.Contains(list, element)
}

// ContainsBy 判断切片是否包含指定元素, 并附带条件
func ContainsBy[T any](list []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(list, predicate)
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

func reverse[T comparable](list []T) {
	length := len(list)
	half := length / 2
	for i := range half {
		j := length - 1 - i
		list[i], list[j] = list[j], list[i]
	}
}

// Reverse 反转切片中的元素, 缺点原地反转
func Reverse[T comparable](list []T) []T {
	reverse(list)
	return list
}

// CloneReverse 反转切片中的元素, 返回新的切片
func CloneReverse[T comparable](list []T) []T {
	data := make([]T, len(list))
	copy(data, list)
	reverse(data)
	return data
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
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		first := true
		for _, item := range source {
			if predicate != nil && !predicate(item) {
				continue
			}
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
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		first := true
		for _, item := range source {
			if predicate != nil && !predicate(item) {
				continue
			}
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
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		for _, item := range source {
			if predicate != nil && !predicate(item) {
				continue
			}
			r += selector(item)
		}
		return
	}
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}

// AvgBy 计算平均值，兼容所有类型
func AvgBy[T comparable, V Integer | Float](q Query[T], selector func(T) V) float64 {
	if q.fastSlice != nil {
		source := q.fastSlice
		predicate := q.fastWhere
		var sum float64
		var n int
		for _, item := range source {
			if predicate != nil && !predicate(item) {
				continue
			}
			sum += float64(selector(item))
			n++
		}
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}
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

// Every 判断子集中的所有元素都包含在集合中
func Every[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	// 子集极大 (M > 100) -> 选哈希
	// 或者list 极大且子集不极小 (N > 2000, M > 50) -> 选哈希
	if m > 100 || n > 2000 && m > 50 {
		return EveryBigData(list, subset)
	}
	// 小规模数据 (NM < 10000) -> 选线性 (无内存分配)
	return EverySmallData(list, subset)
}

// Every 判断子集中的所有元素都包含在集合中 适用于少数据
func EverySmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if !Contains(list, subset[i]) {
			return false
		}
	}
	return true
}

// Every 判断子集中的所有元素都包含在集合中 适用于大数据
func EveryBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 {
		return true
	}
	if len(list) == 0 {
		return false
	}
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

// Some 判断集合中包含子集中的至少有一个元素
func Some[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	if n == 0 || m == 0 {
		return false
	}
	// 如果子集相对较大 (M > 80)
	// 或者在 N 较大时，M 也达到了一定量级 (N*M > 15万 且 M > 30)
	if m > 80 || (n > 5000 && m > 30) {
		return SomeBigData(list, subset)
	}
	// 小规模数据 (NM < 10000) -> 选线性 (无内存分配)
	return SomeSmallData(list, subset)
}

// Some 判断集合中包含子集中的至少有一个元素 适用于少数据
func SomeSmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if Contains(list, subset[i]) {
			return true
		}
	}
	return false
}

// Some 判断集合中包含子集中的至少有一个元素 适用于大数据
func SomeBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 || len(list) == 0 {
		return false
	}
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

// None 判断集合中不包含子集的任何元素
func None[T comparable](list, subset []T) bool {
	n, m := len(list), len(subset)
	if n == 0 || m == 0 {
		return true
	}
	// 如果子集够大，哈希永远是赢家 (复杂度 O(N+M) vs O(N*M))
	// 或者主集合很大，需要子集也有一定规模才值得建表
	if m > 100 || n > 3000 && m > 50 {
		return NoneBigData(list, subset)
	}
	// 小规模或极小子集场景：线性搜索最快且零内存分配
	return NoneSmallData(list, subset)
}

// None 判断集合中不包含子集的任何元素
func NoneSmallData[T comparable](list, subset []T) bool {
	for i := range subset {
		if Contains(list, subset[i]) {
			return false
		}
	}
	return true
}

// None 判断集合中不包含子集的任何元素 适用于大数据
func NoneBigData[T comparable](list []T, subset []T) bool {
	if len(subset) == 0 || len(list) == 0 {
		return true
	}
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

// Union 返回两个切片的并集，自动去重
func Union[T comparable](lists ...[]T) []T {
	var capLen int
	for _, list := range lists {
		capLen += len(list)
	}
	result := make([]T, 0, capLen)
	seen := make(map[T]struct{}, capLen)
	for i := range lists {
		for j := range lists[i] {
			if _, ok := seen[lists[i][j]]; !ok {
				seen[lists[i][j]] = struct{}{}
				result = append(result, lists[i][j])
			}
		}
	}
	return result
}

// Difference 返回两个集合之间的差异, left返回的是list2中不存在的元素的集合, right返回的是list1中不存在的元素的集合
func Difference[T comparable](list1, list2 []T) (left, right []T) {
	seenLeft := map[T]struct{}{}
	seenRight := map[T]struct{}{}
	for i := range list1 {
		seenLeft[list1[i]] = struct{}{}
	}
	for i := range list2 {
		seenRight[list2[i]] = struct{}{}
	}
	for i := range list1 {
		if _, ok := seenRight[list1[i]]; !ok {
			left = append(left, list1[i])
		}
	}
	for i := range list2 {
		if _, ok := seenLeft[list2[i]]; !ok {
			right = append(right, list2[i])
		}
	}
	return left, right
}

// Without 从切片中移除指定的元素
func Without[T comparable](list []T, exclude ...T) []T {
	if len(exclude) == 0 || len(list) == 0 {
		return list
	}
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

// WithoutIndex 从切片中移除指定的索引的元素
func WithoutIndex[T comparable](list []T, index ...int) []T {
	length := len(list)
	if len(index) == 0 || length == 0 {
		return list
	}
	removeSet := make(map[int]struct{}, len(index))
	for i := range index {
		if index[i] >= 0 && index[i] <= length-1 {
			removeSet[index[i]] = struct{}{}
		}
	}
	result := make([]T, 0, len(list))
	for i := range list {
		if _, ok := removeSet[i]; !ok {
			result = append(result, list[i])
		}
	}
	return result
}

// WithoutEmpty 移除切片中的空值（零值）
func WithoutEmpty[T comparable](list []T) []T {
	var empty T
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e != empty {
			result = append(result, e)
		}
	}
	return result
}

// WithoutLEZero 移除切片中小于等于0 的值
func WithoutLEZero[T Float | Integer](list []T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e > 0 {
			result = append(result, e)
		}
	}
	return result
}

// 比较两个列表是否相同
func Equal[T comparable](list1 []T, list2 ...T) bool {
	return EqualBy(list1, list2, func(item T) T { return item })
}

// 比较两个列表是否相同
func EqualBy[T, K comparable](list1, list2 []T, selector func(T) K) bool {
	if len(list1) != len(list2) {
		return false
	}
	if len(list1) == 0 {
		return true
	}
	counters := make(map[K]int, len(list1))
	for _, el := range list1 {
		counters[selector(el)]++
	}
	for _, el := range list2 {
		counters[selector(el)]--
	}
	for _, count := range counters {
		if count != 0 {
			return false
		}
	}
	return true
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
	templist := make([]T, size)
	copy(templist, list)
	results := make([]T, 0, count)
	for i := 0; i < count; i++ {
		remaining := size - i
		index := rand.IntN(remaining)
		results = append(results, templist[index])
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
func Default[T comparable](v T, d ...T) T {
	if len(d) == 0 {
		return Empty[T]()
	}
	if IsEmpty(v) {
		return d[0]
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

// IF 三目运算
func IF[T comparable](cond bool, suc, fail T) T {
	if cond {
		return suc
	} else {
		return fail
	}
}

// Concat 合并多个结果集
func Concat[T comparable](lists ...[]T) []T {
	totalLen := 0
	for i := range lists {
		totalLen += len(lists[i])
	}
	result := make([]T, 0, totalLen)
	for i := range lists {
		result = append(result, lists[i]...)
	}
	return result
}
