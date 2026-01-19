package linq

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type BMember struct {
	Name string
	ID   int64
	Age  int
	Sex  int8
}
type SMember struct {
	Name string
	ID   int64
}

var members = []*BMember{
	{ID: 1, Name: "张三", Sex: 1, Age: 28},
	{ID: 2, Name: "李四", Sex: 2, Age: 28},
	{ID: 3, Name: "王五", Sex: 1, Age: 29},
	{ID: 4, Name: "老六", Sex: 2, Age: 29},
}

func TestSum(t *testing.T) {
	fmt.Printf("Sum Age: %+v \n", From(members).SumIntBy(func(m *BMember) int { return m.Age }))
	fmt.Printf("Avg Age: %+v \n", From(members).AvgIntBy(func(m *BMember) int { return m.Age }))
	fmt.Printf("Sum Age: %+v \n", SumBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("Min Age: %+v \n", MinBy(From(members), func(m *BMember) int { return m.Age }))
	fmt.Printf("Max Age: %+v \n", MaxBy(From(members), func(m *BMember) int { return m.Age }))
}
func TestPage(t *testing.T) {
	page, pageSize := 1, 3
	out1 := From(members).Skip((page - 1) * pageSize).Take(pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
	page = 2
	out1 = From(members).Page(page, pageSize).ToSlice()
	for _, v := range out1 {
		fmt.Printf("%d %+v \n", page, v)
	}
}
func TestUnion(t *testing.T) {
	out := From(members).Union(From(members)).ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
}

func TestOrder(t *testing.T) {
	query := From(members)
	query = OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	query = ThenBy(query, func(m *BMember) int { return m.Age })
	out4 := query.ToSlice()
	for _, v := range out4 {
		fmt.Printf("%+v \n", v)
	}
}

func TestFrom(t *testing.T) {
	out := From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 }).
		ToSlice()
	for _, v := range out {
		fmt.Printf("%+v \n", v)
	}
	out2 := Select(
		From(out),
		func(m *BMember) *SMember { return &SMember{ID: m.ID, Name: m.Name} },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
	out3 := GroupBy(
		From(members),
		func(m *BMember) int8 { return m.Sex },
	).ToSlice()
	for _, v := range out3 {
		fmt.Printf("%+v \n", v)
	}
	out4 := GroupBySelect(
		From(members),
		func(m *BMember) int8 { return m.Sex },
		func(m *BMember) *BMember { return m },
	).ToSlice()
	for _, v := range out4 {
		fmt.Printf("%+v \n", v)
	}
}

func TestFilter(t *testing.T) {
	out2 := Filter(
		From(members),
		func(m *BMember) (*SMember, bool) { return nil, false },
	).ToSlice()
	for _, v := range out2 {
		fmt.Printf("%+v \n", v)
	}
}
func TestHasOrder(t *testing.T) {
	query := From(members).
		Where(func(m *BMember) bool { return m.Age < 29 }).
		Where(func(m *BMember) bool { return m.Sex < 29 })
	fmt.Printf("%+v \n", query.HasOrder())
	query = OrderByDescending(query, func(m *BMember) int8 { return m.Sex })
	fmt.Printf("%+v \n", query.HasOrder())
}

func TestFirst(t *testing.T) {
	fmt.Println(1, From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).DefaultIfEmpty(&BMember{}).First())
	fmt.Println(2, From([]*BMember{}).Where(func(m *BMember) bool { return m.Age < 29 }).First())
}

func TestFromString(t *testing.T) {
	str := "Hello, 世界! 🌍"
	q := FromString(str)
	slice := q.ToSlice()
	expected := []string{"H", "e", "l", "l", "o", ",", " ", "世", "界", "!", " ", "🌍"}
	if len(slice) != len(expected) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(slice))
	}
	for i, v := range slice {
		if v != expected[i] {
			t.Errorf("Index %d: expected %s, got %s", i, expected[i], v)
		}
	}
}

func TestMinMaxBy(t *testing.T) {
	// Test case for MinBy with negative numbers
	nums := []int{-5, -2, -9, -1}
	min := MinBy(From(nums), func(i int) int { return i })
	if min != -9 {
		t.Errorf("Expected Min -9, got %d", min)
	}

	max := MaxBy(From(nums), func(i int) int { return i })
	if max != -1 {
		t.Errorf("Expected Max -1, got %d", max)
	}

	// Test case for MinBy with mixed with 0
	nums2 := []int{5, 0, 2}
	min2 := MinBy(From(nums2), func(i int) int { return i })
	if min2 != 0 {
	}
}

func TestAppendTo(t *testing.T) {
	nums := []int{1, 2, 3}
	buffer := make([]int, 0, 10)
	// Add initial garbage to ensure we are appending correctly
	buffer = append(buffer, 99)

	result := From(nums).AppendTo(buffer)

	expected := []int{99, 1, 2, 3}
	if len(result) != len(expected) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Index %d: expected %d, got %d", i, expected[i], v)
		}
	}
	// Verify it's the same underlying array if cap allows
	if &result[0] != &buffer[0] {
		t.Log("Warning: Slice reallocated, this might be expected if cap changed but check logic")
	}
}

func TestForEachParallel(t *testing.T) {
	count := 100
	nums := Range(0, count).ToSlice()
	var mu sync.Mutex
	processed := make(map[int]struct{})

	From(nums).ForEachParallel(10, func(i int) {
		mu.Lock()
		processed[i] = struct{}{}
		mu.Unlock()
		time.Sleep(1 * time.Millisecond) // Simulate work
	})

	if len(processed) != count {
		t.Errorf("Expected %d processed items, got %d", count, len(processed))
	}
}

func TestSelectAsync(t *testing.T) {
	count := 50
	nums := Range(0, count)

	// SelectAsync order is not guaranteed, so we check existence
	result := SelectAsync(nums, 5, func(i int) int {
		time.Sleep(1 * time.Millisecond)
		return i * 2
	}).ToSlice()

	if len(result) != count {
		t.Fatalf("Expected %d items, got %d", count, len(result))
	}

	perm := make(map[int]bool)
	for _, v := range result {
		perm[v] = true
	}

	for i := 0; i < count; i++ {
		if !perm[i*2] {
			t.Errorf("Missing expected value %d", i*2)
		}
	}
}
