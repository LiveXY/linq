package linq

import (
	"fmt"
	"testing"
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
