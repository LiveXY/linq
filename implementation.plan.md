# Implementation Plan

## 阶段 1：工程结构重构（文件拆分）
- **目标**：将过于庞大（约 3200 行）的 `linq.go` 拆分到不同的职能文件中，降低代码的认知负担。
- **操作指南**：
  1. 创建 `query.go`：包含 `Query` 核心结构、基础构建函数（如 `From`, `FromChannel`, `Range`, `Repeat`）。
  2. 创建 `filter.go`：包含过滤操作（`Where`, `Skip`, `Take`, `SkipWhile`, `TakeWhile`）。
  3. 创建 `aggregate.go`：包含聚合操作（`Count`, `Sum`, `Min`, `Max`, `Average`, `First`, `Last`, `Any`, `All`, `Contains`）。
  4. 创建 `projection.go`：包含映射转换（`Select`, `GroupBy`, `ToMap`, `SelectAsyncCtx`）。
  5. 创建 `set.go`：包含集合操作（`Distinct`, `Union`, `Intersect`, `Except`）。
  6. 创建 `sort.go`：排序操作相关的结构与方法（`OrderBy`, `OrderedQuery`, `Asc`, `Desc` 等）。
  7. 创建 `action.go`：循环和遍历方法（`ForEach`, `ForEachParallel` 等）。
  8. 创建 `utils.go`：内部工具函数。
- **验证**：每次移动代码后通过 `go vet ./...` 和 `go test ./...` 确保重构安全。

## 阶段 2：底座升级：拥抱 Go 1.23 原生 `iter.Seq`
- **目标**：将封闭的 `func() func() (T, bool)` 迭代方式替换为 Go 官方的 `iter.Seq[T]`，全面融入 Go 泛型与迭代器生态。
- **操作指南**：
  1. 引入 `"iter"` 包。
  2. 将 `Query[T]` 中的 `iterate func() func() (T, bool)` 变更为 `iterate iter.Seq[T]`。
  3. 提供 `All() iter.Seq[T]` 供外部在 `for ... range` 中直接调用。
  4. 顺接重写 `From`, `Where`, `Select` 的内部迭代实现。
- **验证**：执行全套单元测试，保证逻辑一致性。

## 阶段 3：类型解放：解除 `comparable` 强约束
- **目标**：突破当前库只能操作具有可较性的类型的限制，允许如 `struct`（含有 slice 等未实现 comparable 成员）使用 LINQ。
- **操作指南**：
  1. 修改 `type Query[T comparable]` 为 `type Query[T any]`。
  2. 剥离依赖了 `comparable` 的原结构体方法，变更为顶层函数，例如将 `q.Distinct()` 变为 `Distinct(q)`。
  3. 为 `KV` 引入半约束 `KV[K comparable, V any]`。
- **验证**：更新测试代码中对链式调用的调用方式，整体编译测试通过。

## 阶段 4：并发安全强化
- **目标**：修复 `SelectAsyncCtx` 等异步方法在 Panic 时的静默崩溃吞并问题以及资源泄漏。
- **操作指南**：
  1. 在 recovering 的时候，通过 channel 传回 err 或者抛出。
  2. 重构内部等待逻辑。

---
我们将按照以上步骤，每完成一个阶段均运行测试确保回归无风险。
