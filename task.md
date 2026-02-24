# Task List

- [ ] **任务 1**: 解析 `linq.go`，梳理所有的函数和方法归属。
- [ ] **任务 2**: 按照实施计划进行文件拆分 (`query.go`, `filter.go`, `aggregate.go`, `projection.go`, `set.go`, `sort.go`, `action.go`)。
- [ ] **任务 3**: 运行测试验证阶段 1 重构无误。
- [ ] **任务 4**: 修改 `Query` 迭代结构为 Go 1.23 `iter.Seq` 并替换所有底层的返回迭代逻辑。
- [ ] **任务 5**: 运行测试验证阶段 2 重构无误。
- [ ] **任务 6**: 将泛型约束从 `comparable` 修改为 `any`，并将无法继承泛型的方法提取为顶层函数。
- [ ] **任务 7**: 升级测试代码以适配新的 API 调用。
- [ ] **任务 8**: 修复 `SelectAsyncCtx` 中的 Goroutine 异常静默问题。
