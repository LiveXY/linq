package linq_benchmark

import "testing"

func SliceSumNormal(list []int) int {
	var sum int
	for _, v := range list {
		sum += v
	}
	return sum
}

func SliceSumUnroll(list []int) int {
	var sum int
	length := len(list)
	i := 0
	for ; i <= length-8; i += 8 {
		sum += list[i] + list[i+1] + list[i+2] + list[i+3] + list[i+4] + list[i+5] + list[i+6] + list[i+7]
	}
	for ; i < length; i++ {
		sum += list[i]
	}
	return sum
}

func BenchmarkSumNormal(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceSumNormal(data)
	}
}

func BenchmarkSumUnroll(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SliceSumUnroll(data)
	}
}
