# LINQ for Go — 高性能泛型查询库

[![Go 1.24+](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://go.dev)

基于 Go 1.24+ 泛型与 `iter.Seq` 迭代器协议的 LINQ 风格查询库，提供 **零分配快速路径 (fastSlice)** 优化，兼顾链式调用的开发体验与极致性能。

## 安装

```bash
go get github.com/livexy/linq
```

## 核心设计

```
┌─────────────┐
│  数据源       │  From / FromChannel / FromString / FromMap / Range / Repeat / Empty
└──────┬──────┘
       ▼
┌─────────────┐
│  Query[T]    │  惰性求值核心结构体
│  ├ fastSlice │  切片快速路径（零迭代器开销）
│  ├ fastWhere │  Where 条件融合（避免多层闭包）
│  └ iterate   │  通用 iter.Seq[T] 迭代器
└──────┬──────┘
       ▼
┌─────────────┐
│  链式操作     │  Where / Select / OrderBy / GroupBy / Distinct / Union / ...
└──────┬──────┘
       ▼
┌─────────────┐
│  终结操作     │  ToSlice / First / Count / Sum / ForEach / ToChannel / ...
└─────────────┘
```

- **惰性求值**：中间操作不会立即执行，直到终结操作触发遍历
- **fastSlice 优化**：`From(slice)` 创建的查询保留底层切片引用，Skip/Take 直接切片运算，Where 条件融合避免多层闭包
- **iter.Seq 协议**：完全兼容 Go 1.23+ 的 `for range` 迭代器

## API 速览

### 创建查询

| 函数 | 说明 |
|------|------|
| `From([]T)` | 从切片创建（启用 fastSlice 优化） |
| `FromChannel(<-chan T)` | 从只读 Channel 创建 |
| `FromString(string)` | 按 UTF-8 字符创建（零拷贝优化） |
| `FromMap(map[K]V)` | 从 Map 创建，元素为 `KV[K, V]` |
| `Range(start, count)` | 创建整数序列 |
| `Repeat(element, count)` | 创建重复元素序列 |
| `Empty[T]()` | 创建空查询 |

### 过滤

| 方法 | 说明 |
|------|------|
| `.Where(predicate)` | 过滤元素（支持条件融合） |
| `.Skip(n)` | 跳过前 N 个元素 |
| `.Take(n)` | 获取前 N 个元素 |
| `.TakeWhile(predicate)` | 连续获取满足条件的元素 |
| `.SkipWhile(predicate)` | 跳过连续满足条件的元素 |
| `.Page(page, pageSize)` | 分页查询 |
| `.DefaultIfEmpty(val)` | 空序列返回默认值 |
| `.Append(item)` | 在末尾追加元素 |
| `.Prepend(item)` | 在开头追加元素 |
| `.Concat(q2)` | 连接两个序列 |

### 投影与变换

| 函数 | 说明 |
|------|------|
| `Select(q, selector)` | 映射每个元素到新类型 |
| `SelectAsync(q, workers, selector)` | 并发映射（无序） |
| `SelectAsyncCtx(ctx, q, workers, selector)` | 并发映射（支持取消） |
| `WhereSelect(q, selector)` | 过滤 + 映射合一 |
| `GroupBy(q, keySelector)` | 按键分组 |
| `GroupBySelect(q, keySelector, elementSelector)` | 分组后映射 |
| `ToMap(q, keySelector)` | 转为 Map |
| `ToMapSelect(q, keySelector, valueSelector)` | 转为 Map（自定义值） |

### 集合操作

| 函数/方法 | 说明 |
|-----------|------|
| `Distinct(q)` / `.Distinct()` | 去重 |
| `DistinctBy(q, selector)` | 按键去重 |
| `Union(q1, q2)` / `.Union(q2)` | 并集 |
| `UnionBy(q1, q2, selector)` | 按键并集 |
| `Intersect(q1, q2)` / `.Intersect(q2)` | 交集 |
| `IntersectBy(q1, q2, selector)` | 按键交集 |
| `Except(q1, q2)` / `.Except(q2)` | 差集 |
| `ExceptBy(q1, q2, selector)` | 按键差集 |
| `DistinctSelect(q, selector)` | 映射 + 去重 |
| `UnionSelect(q, q2, selector)` | 映射 + 并集 |
| `IntersectSelect(q, q2, selector)` | 映射 + 交集 |
| `ExceptSelect(q, q2, selector)` | 映射 + 差集 |

### 排序

| 函数/方法 | 说明 |
|-----------|------|
| `OrderBy(q, key)` | 升序排序 |
| `OrderByDescending(q, key)` | 降序排序 |
| `ThenBy(q, key)` | 次要升序排序 |
| `ThenByDescending(q, key)` | 次要降序排序 |
| `.Order(comparator)` | 自定义排序规则 |
| `.Then(comparator)` | 追加排序规则 |
| `Asc(selector)` | 生成升序比较器 |
| `Desc(selector)` | 生成降序比较器 |
| `.HasOrder()` | 判断是否已定义排序 |
| `.Reverse()` | 反转序列 |

### 聚合与元素访问

| 函数/方法 | 说明 |
|-----------|------|
| `.Count()` / `.CountWith(predicate)` | 计数 |
| `.Any()` / `.AnyWith(predicate)` | 是否存在元素 |
| `.All(predicate)` | 是否全部满足条件 |
| `Sum(q)` / `SumBy(q, selector)` | 求和 |
| `Average(q)` / `AverageBy(q, selector)` | 求平均值 |
| `MinBy(q, selector)` / `MaxBy(q, selector)` | 按选择器取最值（返回元素） |
| `Contains(q, value)` | 是否包含指定元素 |
| `IndexOf(q, value)` / `LastIndexOf(q, value)` | 查找索引 |
| `.IndexOfWith(predicate)` / `.LastIndexOfWith(predicate)` | 按条件查找索引 |
| `.First()` / `.FirstWith(predicate)` | 第一个元素 |
| `.Last()` / `.LastWith(predicate)` | 最后一个元素 |
| `.FirstDefault(defaultValue...)` / `.LastDefault(defaultValue...)` | 带默认值的元素访问 |
| `.Single()` / `.SingleWith(predicate)` / `.SingleDefault(defaultValue...)` | 唯一元素 |

**强类型求和/平均代理**（方法链式调用）：

```go
.SumIntBy(selector)    .SumInt64By(selector)   .SumFloat64By(selector)
.AvgBy(selector)       .AvgIntBy(selector)     .AvgInt64By(selector)
// 以及 int8/int16/int32/uint/uint8/uint16/uint32/uint64/float32 全覆盖
```

### 遍历与并发

| 方法 | 说明 |
|------|------|
| `.ForEach(action)` | 遍历（返回 false 中断） |
| `.ForEachIndexed(action)` | 带索引遍历 |
| `.ForEachParallel(workers, action)` | 并发遍历 |
| `.ForEachParallelCtx(ctx, workers, action)` | 并发遍历（支持 Context 取消） |

### 输出

| 方法 | 说明 |
|------|------|
| `.ToSlice()` | 收集为切片 |
| `.Seq()` | 返回 `iter.Seq[T]` 迭代器 |
| `.ToChannel(ctx)` | 收集为 Channel |
| `.AppendTo(dest)` | 追加到已有切片 |
| `.ToMapSlice(selector)` | 转为 `[]map[string]T` |

### 切片工具函数 (utils.go)

独立于 `Query` 的直接切片操作：

| 函数 | 说明 |
|------|------|
| `Map(list, selector)` / `MapIndexed(list, selector)` | 映射 |
| `Where(list, predicate)` / `WhereIndexed(list, predicate)` | 过滤 |
| `Uniq(list)` | 去重 |
| `SliceContains(list, element)` / `SliceContainsBy(list, pred)` | 包含判断 |
| `SliceIndexOf(list, element)` / `SliceLastIndexOf(list, element)` | 索引查找 |
| `Reverse(list)` / `CloneReverse(list)` | 反转（原地/克隆） |
| `Min(list...)` / `Max(list...)` | 最值 |
| `SliceMinBy(q, selector)` / `SliceMaxBy(q, selector)` | 按选择器取最值 |
| `SliceSumBy(q, selector)` / `SliceAvgBy(q, selector)` | 按选择器求和/平均 |
| `SliceSum(list)` | 切片求和 |
| `Every(list, subset)` / `Some(list, subset)` / `None(list, subset)` | 集合关系判断 |
| `SliceIntersect(a, b)` / `SliceUnion(lists...)` / `Difference(a, b)` | 集合运算 |
| `Without(list, exclude...)` / `WithoutIndex(list, index...)` | 移除元素 |
| `WithoutEmpty(list)` / `WithoutLEZero(list)` | 移除空值/非正值 |
| `Equal(a, b...)` / `EqualBy(a, b, selector)` | 列表比较 |
| `Rand(list, count)` / `Shuffle(list)` | 随机选取/打乱 |
| `Default(v, d...)` / `IsEmpty(v)` / `IsNotEmpty(v)` / `SliceEmpty[T]()` | 零值相关 |
| `IF(cond, suc, fail)` | 三目运算 |
| `Concat(lists...)` | 合并多个切片 |
| `SliceTry(callback, nums...)` / `TryCatch(callback, catch)` | 异常处理 |
| `Try(f)` | 安全执行（返回值 + 错误） |

## 使用示例

### 基础查询链

```go
import "github.com/livexy/linq"

type Member struct {
    Name string
    ID   int64
    Age  int
    Sex  int8
}

members := []*Member{
    {ID: 1, Name: "张三", Sex: 1, Age: 28},
    {ID: 2, Name: "李四", Sex: 2, Age: 28},
    {ID: 3, Name: "王五", Sex: 1, Age: 29},
    {ID: 4, Name: "老六", Sex: 2, Age: 29},
}

// 过滤 + 映射
result := linq.Select(
    linq.From(members).Where(func(m *Member) bool { return m.Age < 29 }),
    func(m *Member) string { return m.Name },
).ToSlice()
// → ["张三", "李四"]
```

### 聚合运算

```go
// 求和
total := linq.From(members).SumIntBy(func(m *Member) int { return m.Age })
// → 114

// 平均值
avg := linq.From(members).AvgIntBy(func(m *Member) int { return m.Age })
// → 28.5

// 最值
youngest := linq.MinBy(linq.From(members), func(m *Member) int { return m.Age })
oldest := linq.MaxBy(linq.From(members), func(m *Member) int { return m.Age })
```

### 排序

```go
// 多字段排序：先按性别降序，再按年龄升序
query := linq.From(members)
query = linq.OrderByDescending(query, func(m *Member) int8 { return m.Sex })
query = linq.ThenBy(query, func(m *Member) int { return m.Age })
sorted := query.ToSlice()

// 或使用 Order + Then 链式排序
sorted2 := linq.From(members).
    Order(linq.Desc(func(m *Member) int8 { return m.Sex })).
    Then(linq.Asc(func(m *Member) int { return m.Age })).
    ToSlice()
```

### 分页

```go
// 第2页，每页3条
page2 := linq.From(members).Page(2, 3).ToSlice()
```

### 分组

```go
// 按性别分组
groups := linq.GroupBy(
    linq.From(members),
    func(m *Member) int8 { return m.Sex },
).ToSlice()
// → [{Key:1, Value:[张三, 王五]}, {Key:2, Value:[李四, 老六]}]
```

### 集合操作

```go
a := linq.From([]int{1, 2, 3, 4})
b := linq.From([]int{3, 4, 5, 6})

union := a.Union(b).ToSlice()        // → [1, 2, 3, 4, 5, 6]
inter := a.Intersect(b).ToSlice()    // → [3, 4]
diff := a.Except(b).ToSlice()        // → [1, 2]
```

### 并发处理

```go
// 并发映射
results := linq.SelectAsync(
    linq.From(urls),
    8,  // 8 个并发 worker
    func(url string) Response { return fetch(url) },
).ToSlice()

// 并发遍历（支持取消）
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
linq.From(tasks).ForEachParallelCtx(ctx, 4, func(task Task) {
    process(task)
})
```

### 迭代器协议

```go
// 直接在 for range 中使用
for item := range linq.From(members).Where(func(m *Member) bool { return m.Age > 28 }).Seq() {
    fmt.Println(item.Name)
}
```

## 性能测试

基于 Apple M4 Pro (macOS/arm64) 的测试结果：

| 测试场景 | 单次耗时 (ns/op) | 内存 (B/op) | 分配次数 (allocs/op) | 说明 |
|---------|-----------------|-------------|---------------------|------|
| `FromString` | **7,757** | **56** | **2** | **零拷贝** UTF-8 解码，内存开销极低 |
| `MinBy` | 16,396 | 72 | 2 | 流式处理，单遍扫描 |
| `Where` | 26,303 | 128,352 | 19 | 10,000 元素过滤 |
| `Union` | 38,573 | 90,648 | 21 | 集合合并（哈希去重） |
| `FromSlice` | 45,833 | 357,697 | 21 | 10,000 元素切片转换 |
| `Select` | 45,879 | 357,729 | 22 | 10,000 元素映射 |
| `Sort` | 10,760 | 50,712 | 32 | 1,000 元素排序 |
| `GroupBy` | 146,036 | 224,864 | 831 | 10,000 元素确定性分组 |

> **亮点**：`FromString` 采用 UTF-8 解码优化，避免全量 `rune` 数组转换；`Where` 条件融合机制避免多层闭包堆叠。

测试命令：
```bash
go test -bench=. -benchmem
```

## 许可证

MIT License
