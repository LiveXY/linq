# Go LINQ 性能对比测试

本项目用于对比不同 Go 语言 LINQ 库（及类 LINQ 库）与原生 Go 实现的性能差异。测试涵盖了常见的过滤（Where）、映射（Select）、链式处理（Chain）、结构体处理、排序及去重等场景。

## 对比对象

1.  **Native**: 原生 Go `for` 循环实现（基准线）。
2.  **Lo (samber/lo)**: 基于泛型的高性能工具库（Eager 模式）。
3.  **LiveXY**: 基于泛型的延迟执行 LINQ 库（Lazy 模式）。
4.  **Ahmetb (go-linq v3)**: 经典的基于 `interface{}` 的 LINQ 库（非泛型）。

## 如何启动测试

在 `linq-test` 目录下运行以下命令即可启动性能基准测试：

```bash
# 运行所有基准测试并统计内存分配
go test -run=^$ -bench . -benchmem ./...
```

如果您希望只测试特定场景（例如 Where），可以使用 `-bench` 参数过滤：

```bash
go test -run=^$ -bench 'Where$' -benchmem ./...
```

建议为了稳定对比，增加重复次数与测试时长，例如：

```bash
go test -run=^$ -bench . -benchmem -count=5 -benchtime=500ms ./...
```

也可以直接使用 Makefile：

```bash
make bench              # 全量性能测试
make bench-where        # 仅 Where 相关
make bench-baseline     # 生成 baseline
make bench-compare      # 与 baseline 做 benchstat 对比
```

## 性能对比结果

测试环境：Apple M4 Pro / darwin arm64  
测试命令：`go test -run=^$ -bench . -benchmem ./...`  
测试时间：2026-03-06

### Query 性能对比（含最优/最差）

| 场景 | 当前实现 | ns/op | 对比最优 | 对比最差 | 结论 |
|---|---|---:|---:|---:|---|
| Where | LiveXYWhere | 267,986 | Lo 74,177 | Ahmetb 1,617,004 | 比最优慢 3.61x，但比最差快 6.03x |
| Select | LiveXYSelect | 147,597 | Lo 51,288 | Ahmetb 2,562,052 | 比最优慢 2.88x，但比最差快 17.36x |
| Chain | LiveXYChain | 148,463 | Lo 106,936 | Ahmetb 2,073,212 | 比最优慢 1.39x，但比最差快 13.96x |
| Struct | LiveXYStruct | 735,754 | Lo 613,064 | Ahmetb 4,154,539 | 比最优慢 1.20x，但比最差快 5.65x |
| Distinct | LiveXYDistinct | 470,080 | Native 442,567 | Ahmetb 2,013,004 | 接近最优（比最优慢 1.06x），但比最差快 4.28x |
| Union | LiveXYUnion | 3,070,700 | LiveXY 3,070,700 | Ahmetb 14,689,322 | 当前最优（比最差快 4.78x） |
| Contains | LiveXYContains | 24,207 | LiveXY 24,207 | Ahmetb 1,312,832 | 当前最优（比最差快 54.23x） |
| Every | LiveXYEvery | 1,208,880 | Lo 1,183,069 | Ahmetb 6,505,001,542 | 接近最优（比最优慢 1.02x），但比最差快 5381.02x |
| Some | LiveXYSome | 1,094 | LiveXY 1,094 | Native 1,150,304 | 当前最优（比最差快 1051.47x） |
| OptimizedSome | LiveXYOptimizedSome | 1,099 | LiveXY 1,099 | Native 1,150,304 | 当前最优（比最差快 1046.68x） |
| None | LiveXYNone | 1,090 | LiveXY 1,090 | Native 1,170,101 | 当前最优（比最差快 1073.49x） |
| Concat | LiveXYConcat | 65,052 | LiveXY 65,052 | Ahmetb 3,912,485 | 当前最优（比最差快 60.14x） |
| Intersect | LiveXYIntersect | 2,808,393 | Native 2,533,547 | Ahmetb 10,208,570 | 比最优慢 1.11x，但比最差快 3.64x |
| Except | LiveXYExcept | 2,355,679 | Native 2,332,963 | Ahmetb 9,923,040 | 接近最优（比最优慢 1.01x），但比最差快 4.21x |
| Reverse | LiveXYReverse | 57,516 | Lo 16,647 | Ahmetb 2,886,447 | 比最优慢 3.46x，但比最差快 50.19x |
| Shuffle | LiveXYShuffle | 606,601 | Lo 564,195 | Native 609,132 | 接近最优（比最优慢 1.08x），但比最差快 1.00x |

### Slice 性能对比（含最优/最差）

| 场景 | 当前实现 | ns/op | 对比最优 | 对比最差 | 结论 |
|---|---|---:|---:|---:|---|
| WhereSlice | LiveXYWhereSlice | 71,569 | LiveXYWhereSlice 71,569 | Ahmetb 1,617,004 | 当前最优（比最差快 22.59x） |
| MapSlice | LiveXYMapSlice | 50,411 | LiveXYMapSlice 50,411 | Ahmetb 2,562,052 | 当前最优（比最差快 50.82x） |
| ChainSlice | LiveXYChainSlice | 108,358 | Lo 106,936 | Ahmetb 2,073,212 | 接近最优（比最优慢 1.01x），但比最差快 19.13x |
| StructSlice | LiveXYStructSlice | 722,713 | Lo 613,064 | Ahmetb 4,154,539 | 比最优慢 1.18x，但比最差快 5.75x |
| SliceUnion | LiveXYSliceUnion | 3,523,960 | LiveXYUnion 3,070,700 | Ahmetb 14,689,322 | 比最优慢 1.15x，但比最差快 4.17x |
| SliceIntersect | LiveXYSliceIntersect | 2,882,879 | Native 2,533,547 | Ahmetb 10,208,570 | 比最优慢 1.14x，但比最差快 3.54x |
| ConcatSlice | LiveXYConcatSlice | 74,826 | LiveXYConcat 65,052 | Ahmetb 3,912,485 | 比最优慢 1.15x，但比最差快 52.29x |
| ReverseSlice | LiveXYReverseSlice | 17,220 | Lo 16,647 | Ahmetb 2,886,447 | 接近最优（比最优慢 1.03x），但比最差快 167.62x |

### 排序性能对比（含稳定/不稳定）

命名说明：`BenchmarkNew*` 为历史基准名，当前分别对应 `Order(...)` / `OrderUnstable(...)` 系列 API。

| 场景 | 当前实现 | ns/op | 对比最优 | 结论 |
|---|---|---:|---:|---|
| 单键(稳定) | LiveXYOneSort | 106,994 | SlicesOneSort 24,262 | 比最优慢 4.41x |
| 单键(稳定) | Order(Asc) [BenchmarkNewOneSort] | 107,953 | SlicesOneSort 24,262 | 比最优慢 4.45x |
| 单键(稳定) | AhmetbOneSort | 125,792 | SlicesOneSort 24,262 | 比最优慢 5.18x |
| 单键(稳定) | NativeOneSort | 33,727 | SlicesOneSort 24,262 | 比最优慢 1.39x |
| 单键(稳定) | SlicesOneSort | 24,262 | SlicesOneSort 24,262 | 当前最优 |
| 单键(不稳定) | LiveXYOneSortUnstable | 43,038 | SlicesOneSort 24,262 | 比最优慢 1.77x |
| 单键(不稳定) | OrderUnstable(Asc) [BenchmarkNewOneSortUnstable] | 45,325 | SlicesOneSort 24,262 | 比最优慢 1.87x |
| 双键(稳定) | LiveXYTwoSort | 139,498 | SlicesTwoSort 29,435 | 比最优慢 4.74x |
| 双键(稳定) | Order+Then [BenchmarkNewTwoSort] | 141,765 | SlicesTwoSort 29,435 | 比最优慢 4.82x |
| 双键(稳定) | AhmetbTwoSort | 157,771 | SlicesTwoSort 29,435 | 比最优慢 5.36x |
| 双键(稳定) | NativeTwoSort | 41,747 | SlicesTwoSort 29,435 | 比最优慢 1.42x |
| 双键(稳定) | SlicesTwoSort | 29,435 | SlicesTwoSort 29,435 | 当前最优 |
| 双键(不稳定) | LiveXYTwoSortUnstable | 79,403 | SlicesTwoSort 29,435 | 比最优慢 2.70x |
| 双键(不稳定) | OrderUnstable+Then [BenchmarkNewTwoSortUnstable] | 84,070 | SlicesTwoSort 29,435 | 比最优慢 2.86x |
| 三键(稳定) | LiveXYThreeSort | 147,576 | SlicesStableFunc 86,901 | 比最优慢 1.70x |
| 三键(稳定) | Order+Then+Then [BenchmarkNewThreeSort] | 148,071 | SlicesStableFunc 86,901 | 比最优慢 1.70x |
| 三键(稳定) | Order(单比较器) [BenchmarkNew2ThreeSort] | 90,116 | SlicesStableFunc 86,901 | 接近最优（比最优慢 1.04x） |
| 三键(稳定) | AhmetbThreeSort | 198,602 | SlicesStableFunc 86,901 | 比最优慢 2.29x |
| 三键(稳定) | NativeStableThreeSort | 217,985 | SlicesStableFunc 86,901 | 比最优慢 2.51x |
| 三键(稳定) | SlicesStableFuncThreeSort | 86,901 | SlicesStableFunc 86,901 | 当前最优 |
| 三键(稳定) | SlicesStableCompareThreeSort | 87,061 | SlicesStableFunc 86,901 | 接近最优（比最优慢 1.00x） |
| 三键(不稳定) | LiveXYThreeSortUnstable | 98,161 | SlicesSortFunc 38,179 | 比最优慢 2.57x |
| 三键(不稳定) | OrderUnstable+Then+Then [BenchmarkNewThreeSortUnstable] | 105,858 | SlicesSortFunc 38,179 | 比最优慢 2.77x |
| 三键(不稳定) | OrderUnstable(单比较器) [BenchmarkNew2ThreeSortUnstable] | 40,867 | SlicesSortFunc 38,179 | 接近最优（比最优慢 1.07x） |
| 三键(不稳定) | NativeThreeSort | 56,044 | SlicesSortFunc 38,179 | 比最优慢 1.47x |
| 三键(不稳定) | SlicesSortFuncThreeSort | 38,179 | SlicesSortFunc 38,179 | 当前最优 |
| 三键(不稳定) | SlicesSortCompareThreeSort | 38,399 | SlicesSortFunc 38,179 | 接近最优（比最优慢 1.01x） |

### 稳定 vs 不稳定收益对照（按提速倍率排序）

| 对比项 | 稳定 ns/op | 不稳定 ns/op | 提速倍率 | 绝对减少(ns) | 结论 |
|---|---:|---:|---:|---:|---|
| LiveXYOneSort | 106,994 | 43,038 | 2.49x | 63,956 | 收益最高，优先使用不稳定排序 |
| Order(Asc) [BenchmarkNewOneSort] | 107,953 | 45,325 | 2.38x | 62,628 | 收益很高，明显优于稳定版 |
| Order(单比较器) [BenchmarkNew2ThreeSort] | 90,116 | 40,867 | 2.21x | 49,249 | 收益很高，接近原生最快组 |
| LiveXYTwoSort | 139,498 | 79,403 | 1.76x | 60,095 | 收益显著，双键场景建议优先不稳定 |
| Order+Then [BenchmarkNewTwoSort] | 141,765 | 84,070 | 1.69x | 57,695 | 收益显著，链式 Then 同样受益 |
| LiveXYThreeSort | 147,576 | 98,161 | 1.50x | 49,415 | 中等收益，三键链路有改善 |
| Order+Then+Then [BenchmarkNewThreeSort] | 148,071 | 105,858 | 1.40x | 42,213 | 中等收益，仍建议优先不稳定 |

## 结论摘要

- `Select` 主链相较上一轮有明显改善（约 `307,649 -> 147,597 ns/op`），`Some/None` 进入 `~1,100 ns/op`。
- `Union/Contains/Some/OptimizedSome/None/Concat` 继续保持最优，`Except` 已接近 `Native`（约 `1.01x`）。
- 当前主要短板是 Query 主链 `Where/Reverse` 与稳定排序链路；不稳定排序可持续显著缩小差距。
