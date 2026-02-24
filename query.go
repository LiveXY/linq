package linq

import (
	"context"
	"iter"
	"maps"
	"slices"
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

// KV 键值对结构体
type KV[K comparable, V any] struct {
	Key   K
	Value V
}

// CompareFunc 比较函数类型
type CompareFunc[T any] func(a, b T) int

// Query 查询结构体，是 LINQ 操作的核心类型
type Query[T any] struct {
	compare    CompareFunc[T]
	iterate    iter.Seq[T]
	fastSlice  []T
	fastWhere  func(T) bool
	capacity   int
	sortSource *Query[T]
}

// Seq 返回供 for-range 从头到尾遍历的迭代器
func (q Query[T]) Seq() iter.Seq[T] {
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
	return slices.Collect(q.iterate)
}

// ToChannel 将查询结果收集为通道，支持上下文取消
func (q Query[T]) ToChannel(ctx context.Context) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
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

// From 从切片创建 Query 查询对象
func From[T any](source []T) Query[T] {
	return Query[T]{
		iterate:   slices.Values(source),
		fastSlice: source,
		capacity:  len(source),
	}
}

// FromChannel 从只读 Channel 创建 Query 查询对象
func FromChannel[T any](source <-chan T) Query[T] {
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for item := range source {
				if !yield(item) {
					break
				}
			}
		},
	}
}

// FromString 从字符串创建 Query 查询对象，每个元素为一个 UTF-8 字符
func FromString(source string) Query[string] {
	return Query[string]{
		iterate: func(yield func(string) bool) {
			pos := 0
			length := len(source)
			for pos < length {
				r, w := utf8.DecodeRuneInString(source[pos:])
				var item string
				if r == utf8.RuneError && w == 1 {
					item = string(r)
				} else {
					item = source[pos : pos+w]
				}
				pos += w
				if !yield(item) {
					break
				}
			}
		},
		capacity: len(source),
	}
}

// FromMap 从 Map 创建 Query 查询对象，每个元素为 KV 键值对
func FromMap[K comparable, V any](source map[K]V) Query[KV[K, V]] {
	return Query[KV[K, V]]{
		iterate: func(yield func(KV[K, V]) bool) {
			for k, v := range maps.All(source) {
				if !yield(KV[K, V]{Key: k, Value: v}) {
					break
				}
			}
		},
		capacity: len(source),
	}
}

// Empty 创建一个空的 Query 查询对象
func Empty[T any]() Query[T] {
	return From([]T{})
}

// Range 创建一个包含指定范围内整数序列的 Query 查询对象
func Range(start, count int) Query[int] {
	if count <= 0 {
		return Empty[int]()
	}
	return Query[int]{
		iterate: func(yield func(int) bool) {
			end := start + count
			for i := start; i < end; i++ {
				if !yield(i) {
					return
				}
			}
		},
		capacity: count,
	}
}

// Repeat 创建一个包含重复元素的 Query 查询对象
func Repeat[T any](element T, count int) Query[T] {
	if count <= 0 {
		return Empty[T]()
	}
	return Query[T]{
		iterate: func(yield func(T) bool) {
			for i := 0; i < count; i++ {
				if !yield(element) {
					return
				}
			}
		},
		capacity: count,
	}
}
