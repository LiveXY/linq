package linq

import (
	"fmt"
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
		t.Errorf("期望 99 个处理项，实际得到 %d", processed.Load())
	}
	if panicked.Load() != 1 {
		t.Errorf("期望 1 次 panic，实际得到 %d", panicked.Load())
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
		t.Fatalf("期望 10 个元素，实际得到 %d", len(result))
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
		t.Errorf("期望 99 个元素，实际得到 %d", len(result))
	}
}

// 测试 BufferPool
func TestBufferPool(t *testing.T) {
	pool := NewBufferPool[int]()

	// 获取 buffer
	buf1 := pool.Get(100)
	if cap(buf1) < 100 {
		t.Errorf("期望容量 >= 100，实际得到 %d", cap(buf1))
	}

	// 使用 buffer
	buf1 = append(buf1, 1, 2, 3)

	// 归还 buffer
	pool.Put(buf1)

	// 再次获取，应该复用
	buf2 := pool.Get(50)
	if len(buf2) != 0 {
		t.Errorf("期望空 buffer，实际长度 %d", len(buf2))
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
		t.Errorf("期望 100 个不重复元素，实际得到 %d", len(result))
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
		t.Errorf("期望 50 个元素，实际得到 %d", len(result))
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

// 测试空集合的 Avg 不会返回 NaN
func TestAvgEmptySlice(t *testing.T) {
	empty := []int{}
	avg := From(empty).AvgIntBy(func(i int) int { return i })
	if avg != 0 {
		t.Errorf("空切片时期望 0，实际得到 %f", avg)
	}

	avg64 := From(empty).AvgInt64By(func(i int) int64 { return int64(i) })
	if avg64 != 0 {
		t.Errorf("空切片时期望 0，实际得到 %f", avg64)
	}

	avgFloat := From(empty).AvgBy(func(i int) float64 { return float64(i) })
	if avgFloat != 0 {
		t.Errorf("空切片时期望 0，实际得到 %f", avgFloat)
	}
}

// 测试 Filter 正确过滤多个元素
func TestFilterMultiple(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 只保留偶数并转换为字符串
	result := Filter(From(nums), func(i int) (string, bool) {
		if i%2 == 0 {
			return fmt.Sprintf("%d", i), true
		}
		return "", false
	}).ToSlice()

	expected := []string{"2", "4", "6", "8", "10"}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %s，实际得到 %s", i, expected[i], v)
		}
	}
}

// 测试 Without 性能优化后的正确性
func TestWithoutOptimized(t *testing.T) {
	list := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := Without(list, 2, 4, 6, 8, 10)

	expected := []int{1, 3, 5, 7, 9}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("索引 %d: 期望 %d，实际得到 %d", i, expected[i], v)
		}
	}
}
