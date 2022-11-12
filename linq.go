package linq

import (
	"math/rand"
	"time"

	"golang.org/x/exp/constraints"
)

type lesserFunc[T any] func([]T) func(i, j int) bool

type KV[K comparable, V any] struct {
	Key   K
	Value V
}

type Query[T any] struct {
	lesser  lesserFunc[T]
	iterate func() func() (T, bool)
}

func From[T any](source []T) Query[T] {
	len := len(source)
	return Query[T]{
		iterate: func() func() (T, bool) {
			index := 0
			return func() (item T, ok bool) {
				ok = index < len
				if ok {
					item = source[index]
					index++
				}
				return
			}
		},
	}
}
func FromChannel[T any](source <-chan T) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			return func() (item T, ok bool) {
				item, ok = <-source
				return
			}
		},
	}
}
func FromString(source string) Query[string] {
	runes := []rune(source)
	len := len(runes)
	return Query[string]{
		iterate: func() func() (string, bool) {
			index := 0
			return func() (item string, ok bool) {
				ok = index < len
				if ok {
					item = string(runes[index])
					index++
				}
				return
			}
		},
	}
}
func FromMap[K comparable, V any](source map[K]V) Query[KV[K, V]] {
	len := len(source)
	keyvalues := make([](KV[K, V]), 0, len)
	for key, value := range source {
		keyvalues = append(keyvalues, KV[K, V]{Key: key, Value: value})
	}
	return From(keyvalues)
}

func (q Query[T]) Where(predicate func(T) bool) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if predicate(item) {
						return
					}
				}
				return
			}
		},
	}
}
func (q Query[T]) Skip(count int) Query[T] {
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
func (q Query[T]) Take(count int) Query[T] {
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
func (q Query[T]) Page(page, pageSize int) Query[T] {
	return q.Skip((page - 1) * pageSize).Take(pageSize)
}
func (q Query[T]) Union(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
func (q Query[T]) Append(item T) Query[T] {
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
func (q Query[T]) Concat(q2 Query[T]) Query[T] {
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
func (q Query[T]) Prepend(item T) Query[T] {
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
func (q Query[T]) DefaultIfEmpty(defaultValue T) Query[T] {
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
func (q Query[T]) Distinct() Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			set := make(map[any]struct{})
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
func (q Query[T]) Except(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
func (q Query[T]) IndexOf(predicate func(T) bool) int {
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
func (q Query[T]) Intersect(q2 Query[T]) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
func (q Query[T]) All(predicate func(T) bool) bool {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !predicate(item) {
			return false
		}
	}
	return true
}
func (q Query[T]) Any() bool {
	_, ok := q.iterate()()
	return ok
}
func (q Query[T]) AnyWith(predicate func(T) bool) bool {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			return true
		}
	}
	return false
}
func (q Query[T]) CountWith(predicate func(T) bool) (r int) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			r++
		}
	}
	return
}
func (q Query[T]) First() T {
	item, _ := q.iterate()()
	return item
}
func (q Query[T]) FirstWith(predicate func(T) bool) T {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			return item
		}
	}
	var out T
	return out
}
func (q Query[T]) ForEach(action func(T) bool) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if !action(item) {
			return
		}
	}
}
func (q Query[T]) ForEachIndexed(action func(int, T) bool) {
	next := q.iterate()
	index := 0
	for item, ok := next(); ok; item, ok = next() {
		if !action(index, item) {
			return
		}
		index++
	}
}
func (q Query[T]) Last() (r T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = item
	}
	return
}
func (q Query[T]) LastWith(predicate func(T) bool) (r T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		if predicate(item) {
			r = item
		}
	}
	return
}
func (q Query[T]) Reverse() Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			next := q.iterate()
			var items []T
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
	}
}
func (q Query[T]) Single() (r T) {
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

func (q Query[T]) SumIntBy(selector func(T) int64) (r int64) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) SumFloatBy(selector func(T) float64) (r float64) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r += selector(item)
	}
	return
}
func (q Query[T]) AvgBy(selector func(T) float64) float64 {
	next := q.iterate()
	var sum float64
	var n int
	for item, ok := next(); ok; item, ok = next() {
		sum += selector(item)
		n++
	}
	return float64(sum) / float64(n)
}

func (q Query[T]) Count() (r int) {
	next := q.iterate()
	for _, ok := next(); ok; _, ok = next() {
		r++
	}
	return
}
func (q Query[T]) ToSlice() (r []T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, item)
	}
	return
}
func (q Query[T]) ToChannel(c chan<- T) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		c <- item
	}
	close(c)
}
func (q Query[T]) ToMap(selector func(T) map[string]any) (r []map[string]any) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		r = append(r, selector(item))
	}
	return
}

func GroupBy[T any, K comparable](q Query[T], keySelector func(T) K) Query[KV[K, []T]] {
	return Query[KV[K, []T]]{
		iterate: func() func() (KV[K, []T], bool) {
			next := q.iterate()
			set := make(map[K][]T)
			for item, ok := next(); ok; item, ok = next() {
				key := keySelector(item)
				set[key] = append(set[key], item)
			}
			len := len(set)
			idx := 0
			groups := make([](KV[K, []T]), len)
			for k, v := range set {
				groups[idx] = KV[K, []T]{k, v}
				idx++
			}
			index := 0
			return func() (item KV[K, []T], ok bool) {
				ok = index < len
				if ok {
					item = groups[index]
					index++
				}
				return
			}
		},
	}
}
func GroupBySelect[T any, K comparable, V any](q Query[T], keySelector func(T) K, elementSelector func(T) V) Query[KV[K, []V]] {
	return Query[KV[K, []V]]{
		iterate: func() func() (KV[K, []V], bool) {
			next := q.iterate()
			set := make(map[K][]V)
			for item, ok := next(); ok; item, ok = next() {
				key := keySelector(item)
				set[key] = append(set[key], elementSelector(item))
			}
			len := len(set)
			idx := 0
			groups := make([](KV[K, []V]), len)
			for k, v := range set {
				groups[idx] = KV[K, []V]{k, v}
				idx++
			}
			index := 0
			return func() (item KV[K, []V], ok bool) {
				ok = index < len
				if ok {
					item = groups[index]
					index++
				}
				return
			}
		},
	}
}
func Select[T, V any](q Query[T], selector func(T) V) Query[V] {
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
	}
}
func Filter[T, V any](q Query[T], selector func(T) (V, bool)) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			return func() (item V, ok bool) {
				var it T
				it, ok = next()
				if ok {
					item, ok = selector(it)
				}
				return
			}
		},
	}
}
func Distinct[T, V any](q Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			set := make(map[any]struct{})
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; !has {
						set[s] = struct{}{}
						return
					}
				}
				return
			}
		},
	}
}
func ExceptBy[T, V any](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
			for i, ok := next2(); ok; i, ok = next2() {
				s := selector(i)
				set[s] = struct{}{}
			}
			return func() (item V, ok bool) {
				var it T
				for it, ok = next(); ok; it, ok = next() {
					s := selector(it)
					if _, has := set[s]; !has {
						return
					}
				}
				return
			}
		},
	}
}
func Range[T constraints.Integer](start, count T) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index T
			current := start
			var it T
			return func() (item T, ok bool) {
				if index >= count {
					return it, false
				}
				item, ok = current, true
				index++
				current++
				return
			}
		},
	}
}
func Repeat[T constraints.Ordered](value T, count int) Query[T] {
	return Query[T]{
		iterate: func() func() (T, bool) {
			var index int
			var it T
			return func() (item T, ok bool) {
				if index >= count {
					return it, false
				}
				item, ok = value, true
				index++
				return
			}
		},
	}
}
func IntersectBy[T, V any](q Query[T], q2 Query[T], selector func(T) V) Query[V] {
	return Query[V]{
		iterate: func() func() (V, bool) {
			next := q.iterate()
			next2 := q2.iterate()
			set := make(map[any]struct{})
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
						return
					}
				}
				return
			}
		},
	}
}
func ToMap[T, K comparable](q Query[T], selector func(T) K) map[K]T {
	ret := make(map[K]T)
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		k := selector(item)
		ret[k] = item
	}
	return ret
}

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
func Contains[T comparable](list []T, element T) bool {
	for _, item := range list {
		if item == element {
			return true
		}
	}
	return false
}
func IndexOf[T comparable](list []T, element T) int {
	for i, item := range list {
		if item == element {
			return i
		}
	}
	return -1
}
func LastIndexOf[T comparable](list []T, element T) int {
	length := len(list)
	for i := length - 1; i >= 0; i-- {
		if list[i] == element {
			return i
		}
	}
	return -1
}
func Shuffle[T any](list []T) []T {
	rand.Shuffle(len(list), func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})
	return list
}
func Reverse[T any](list []T) []T {
	length := len(list)
	half := length / 2
	for i := 0; i < half; i = i + 1 {
		j := length - 1 - i
		list[i], list[j] = list[j], list[i]
	}
	return list
}
func Min[T constraints.Ordered](list ...T) T {
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
func Max[T constraints.Ordered](list ...T) T {
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
func MinBy[T any, V constraints.Integer](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		n := selector(item)
		if r > n {
			r = n
		} else {
			if r == 0 && n != r {
				r = n
			}
		}
	}
	return
}
func MaxBy[T any, V constraints.Integer](q Query[T], selector func(T) V) (r V) {
	next := q.iterate()
	for item, ok := next(); ok; item, ok = next() {
		n := selector(item)
		if r < n {
			r = n
		} else {
			if r == 0 && n != r {
				r = n
			}
		}
	}
	return
}

func Sum[T constraints.Float | constraints.Integer | constraints.Complex](list []T) T {
	var sum T = 0
	for _, val := range list {
		sum += val
	}
	return sum
}
func Every[T comparable](list []T, subset []T) bool {
	for _, elem := range subset {
		if !Contains(list, elem) {
			return false
		}
	}
	return true
}
func Some[T comparable](list []T, subset []T) bool {
	for _, elem := range subset {
		if Contains(list, elem) {
			return true
		}
	}
	return false
}
func None[T comparable](list []T, subset []T) bool {
	for _, elem := range subset {
		if Contains(list, elem) {
			return false
		}
	}
	return true
}
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
func Difference[T comparable](list1 []T, list2 []T) ([]T, []T) {
	left := []T{}
	right := []T{}
	seenLeft := map[T]struct{}{}
	seenRight := map[T]struct{}{}
	for _, elem := range list1 {
		seenLeft[elem] = struct{}{}
	}
	for _, elem := range list2 {
		seenRight[elem] = struct{}{}
	}
	for _, elem := range list1 {
		if _, ok := seenRight[elem]; !ok {
			left = append(left, elem)
		}
	}
	for _, elem := range list2 {
		if _, ok := seenLeft[elem]; !ok {
			right = append(right, elem)
		}
	}
	return left, right
}
func Union[T comparable](list1 []T, list2 []T) []T {
	result := []T{}
	seen := map[T]struct{}{}
	hasAdd := map[T]struct{}{}
	for _, e := range list1 {
		seen[e] = struct{}{}
	}
	for _, e := range list2 {
		seen[e] = struct{}{}
	}
	for _, e := range list1 {
		if _, ok := seen[e]; ok {
			result = append(result, e)
			hasAdd[e] = struct{}{}
		}
	}
	for _, e := range list2 {
		if _, ok := hasAdd[e]; ok {
			continue
		}
		if _, ok := seen[e]; ok {
			result = append(result, e)
		}
	}
	return result
}
func Without[T comparable](list []T, exclude ...T) []T {
	result := make([]T, 0, len(list))
	for _, e := range list {
		if !Contains(exclude, e) {
			result = append(result, e)
		}
	}
	return result
}
func NoEmpty[T comparable](list []T) []T {
	var empty T
	result := make([]T, 0, len(list))
	for _, e := range list {
		if e != empty {
			result = append(result, e)
		}
	}
	return result
}
func Rand[T any](list []T, count int) []T {
	size := len(list)
	templist := append([]T{}, list...)
	results := []T{}
	for i := 0; i < size && i < count; i++ {
		copyLength := size - i
		index := rand.Intn(size - i)
		results = append(results, templist[index])
		templist[index] = templist[copyLength-1]
		templist = templist[:copyLength-1]
	}
	return results
}
func Default[T comparable](v, d T) T {
	if IsEmpty(v) {
		return d
	}
	return v
}
func Empty[T any]() T {
	var zero T
	return zero
}
func IsEmpty[T comparable](v T) bool {
	var zero T
	return zero == v
}
func IsNotEmpty[T comparable](v T) bool {
	var zero T
	return zero != v
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
func TryCatch(callback func() error, catch func()) {
	if !try(callback) {
		catch()
	}
}
