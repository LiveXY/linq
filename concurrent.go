package linq

import (
	"sync"
)

// BufferPool 提供切片复用，减少 GC 压力
// 使用示例:
//
//	buf := BufferPool.Get(1000)
//	result := From(data).AppendTo(buf)
//	// 使用完后归还
//	BufferPool.Put(result[:0])
type bufferPool[T any] struct {
	pool sync.Pool
}

func (p *bufferPool[T]) Get(capacity int) []T {
	if v := p.pool.Get(); v != nil {
		buf := v.([]T)
		if cap(buf) >= capacity {
			return buf[:0]
		}
	}
	return make([]T, 0, capacity)
}

func (p *bufferPool[T]) Put(buf []T) {
	if cap(buf) > 0 {
		p.pool.Put(buf[:0])
	}
}

// NewBufferPool 创建一个新的 buffer pool
func NewBufferPool[T any]() *bufferPool[T] {
	return &bufferPool[T]{}
}

// DistinctComparable 为 comparable 类型提供优化的去重实现，避免装箱
func DistinctComparable[T comparable](q Query[T]) Query[T] {
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

// ExceptComparable 为 comparable 类型提供优化的差集实现
func ExceptComparable[T comparable](q Query[T], q2 Query[T]) Query[T] {
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

// IntersectComparable 为 comparable 类型提供优化的交集实现
func IntersectComparable[T comparable](q Query[T], q2 Query[T]) Query[T] {
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
