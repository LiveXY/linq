package linq

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 测试 ForEachParallel 的 panic 恢复
func TestForEachParallelPanicRecovery(t *testing.T) {
	nums := Range(0, 100).ToSlice()
	var processed atomic.Int32
	var panicked atomic.Int32

	From(nums).ForEachParallel(10, func(i int) {
		if i == 50 {
			panicked.Add(1)
			panic("test panic")
		}
		processed.Add(1)
		time.Sleep(1 * time.Millisecond)
	})

	// 应该处理了除了 panic 的那个之外的所有元素
	if processed.Load() != 99 {
		t.Errorf("Expected 99 processed items, got %d", processed.Load())
	}
	if panicked.Load() != 1 {
		t.Errorf("Expected 1 panic, got %d", panicked.Load())
	}
}

// 测试 SelectAsync 提前退出不会导致 goroutine 泄漏
func TestSelectAsyncEarlyExit(t *testing.T) {
	count := 1000
	nums := Range(0, count)

	// 只取前 10 个结果
	result := SelectAsync(nums, 10, func(i int) int {
		time.Sleep(10 * time.Millisecond) // 模拟耗时操作
		return i * 2
	}).Take(10).ToSlice()

	if len(result) != 10 {
		t.Fatalf("Expected 10 items, got %d", len(result))
	}

	// 等待一段时间，确保后台 goroutine 能够正常退出
	time.Sleep(100 * time.Millisecond)

	// 注意：这里无法直接检测 goroutine 泄漏，但可以通过 runtime.NumGoroutine()
	// 在实际使用中观察
}

// 测试 SelectAsync 的 panic 恢复
func TestSelectAsyncPanicRecovery(t *testing.T) {
	count := 100
	nums := Range(0, count)

	result := SelectAsync(nums, 5, func(i int) int {
		if i == 50 {
			panic("test panic in selector")
		}
		return i * 2
	}).ToSlice()

	// 应该得到除了 panic 的那个之外的所有结果
	if len(result) != 99 {
		t.Errorf("Expected 99 items, got %d", len(result))
	}
}

// 测试 BufferPool
func TestBufferPool(t *testing.T) {
	pool := NewBufferPool[int]()

	// 获取 buffer
	buf1 := pool.Get(100)
	if cap(buf1) < 100 {
		t.Errorf("Expected capacity >= 100, got %d", cap(buf1))
	}

	// 使用 buffer
	buf1 = append(buf1, 1, 2, 3)

	// 归还 buffer
	pool.Put(buf1)

	// 再次获取，应该复用
	buf2 := pool.Get(50)
	if len(buf2) != 0 {
		t.Errorf("Expected empty buffer, got length %d", len(buf2))
	}
}

// 测试 DistinctComparable 性能
func TestDistinctComparable(t *testing.T) {
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i % 100 // 100 个不同的值
	}

	result := DistinctComparable(From(nums)).ToSlice()

	if len(result) != 100 {
		t.Errorf("Expected 100 distinct items, got %d", len(result))
	}
}

// 基准测试：对比 Distinct 和 DistinctComparable
func BenchmarkDistinct(b *testing.B) {
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i % 1000
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		From(nums).Distinct().ToSlice()
	}
}

func BenchmarkDistinctComparable(b *testing.B) {
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i % 1000
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DistinctComparable(From(nums)).ToSlice()
	}
}

// 测试 AppendTo 与 BufferPool 结合使用
func TestAppendToWithPool(t *testing.T) {
	pool := NewBufferPool[int]()
	nums := Range(0, 100).ToSlice()

	// 从 pool 获取 buffer
	buf := pool.Get(100)

	// 使用 AppendTo
	result := From(nums).Where(func(i int) bool { return i%2 == 0 }).AppendTo(buf)

	if len(result) != 50 {
		t.Errorf("Expected 50 items, got %d", len(result))
	}

	// 归还 buffer
	pool.Put(result)
}

// 并发安全测试
func TestConcurrentBufferPool(t *testing.T) {
	pool := NewBufferPool[int]()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := pool.Get(10)
			buf = append(buf, 1, 2, 3)
			time.Sleep(1 * time.Millisecond)
			pool.Put(buf)
		}()
	}

	wg.Wait()
}
