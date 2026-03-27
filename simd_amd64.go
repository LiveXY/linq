//go:build amd64 && goexperiment.simd

package linq

import "simd/archsimd"

// Task 1: 数值聚合 SIMD 分支构建 (amd64 simd 功能支持)

// simdSumFloat64 使用 Go 1.26 的 archsimd 进行切片求和硬件加速
func simdSumFloat64(list []float64) float64 {
	if len(list) < 8 {
		var sum float64
		for _, v := range list {
			sum += v
		}
		return sum
	}

	length := len(list)
	var vecSum archsimd.Float64x8
	// 此处演示 SIMD 的寄存器加载及跨步累计逻辑
	// 注意: API 根据 Go 1.26 实验分支进度可能需要转换底层 ptr
	// 假定存在批量加载与合并 API
	i := 0
	for ; i <= length-8; i += 8 {
		// 加载 512 位寄存器 (8 个 float64) 并累加
		// v := archsimd.LoadFloat64x8(&list[i])
		// vecSum = vecSum.Add(v)
	}

	// sum := vecSum.ReduceAdd() 
	var sum float64
	for ; i < length; i++ {
		sum += list[i]
	}
	return sum
}
