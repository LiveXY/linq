package linq

import (
	"time"
)

// IF 三目运算
func IF[T comparable](cond bool, suc, fail T) T {
	if cond {
		return suc
	} else {
		return fail
	}
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

// TryDelay 尝试执行函数，支持重试和延迟
func TryDelay(callback func() error, nums ...int) bool {
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
	if !TryDelay(callback) {
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
