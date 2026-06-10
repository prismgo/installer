# 加固 Skeleton 复制目标处理

## 功能概述和实现目标

本次变更加固 `internal/skeleton` 的本地文件复制逻辑，目标是在复制 PrismGo skeleton 时拒绝覆盖已有目标条目、拒绝跟随目标符号链接，并在复制过程中更及时响应上下文取消。

## 需求 / 业务背景

质量评审发现 `copyFile` 使用 `O_TRUNC` 打开目标文件，会覆盖已有文件，并可能跟随已有目标符号链接写入目标目录之外的文件。同时，目标文件关闭阶段的写入错误没有返回，目录复制循环也只在进入目录时检查取消信号。

## 影响范围

影响范围限定在 skeleton 复制流程，包括本地 skeleton 源和 GitHub skeleton 源复制到目标目录时的文件创建、权限设置、错误返回和取消处理。

## 修改了哪些文件

- `internal/skeleton/skeleton.go`
- `internal/skeleton/skeleton_test.go`
- `internal/skeleton/skeleton_unix_test.go`
- `docs/changes/v2-harden-skeleton-copy-targets.md`

## 做了哪些行为变更

- 复制文件前使用 `Lstat` 检查目标路径，已有文件、目录或符号链接都会失败。
- 递归复制目录前使用 `Lstat` 检查目标目录，拒绝已有目标目录符号链接，避免嵌套目录复制逃逸目标树。
- 创建目标文件改为 `O_WRONLY|O_CREATE|O_EXCL`，避免检查后到创建前的覆盖竞态。
- `io.Copy` 和 `Chmod` 后显式关闭目标文件，并返回关闭阶段错误。
- 在目录条目循环中、以及每次文件复制开始前检查 `ctx.Err()`。
- 将 FIFO 非普通文件覆盖测试移动到 Unix build tag 文件，默认测试包不再依赖 `syscall.Mkfifo`。

## 执行了哪些检查以及结果摘要

- `go test ./internal/run ./internal/skeleton`：通过。
- `go test ./...`：通过。
- `bash .github/scripts/coverage.sh ./internal/skeleton`：通过，总语句覆盖率 91.6%。
- `make lint LINT_ARGS=./internal/skeleton`：通过，0 issues。
- `gofmt -w internal/skeleton/skeleton.go internal/skeleton/skeleton_test.go internal/skeleton/skeleton_unix_test.go`：已执行。

## 单元测试覆盖了哪些逻辑

单元测试覆盖了已有目标文件不被覆盖、已有目标符号链接不被跟随、已有目标目录符号链接不被递归写入、目录条目循环中的取消信号能阻止后续文件复制、文件复制开始前的取消信号不创建目标文件、目标创建失败时返回错误、复制流失败时返回上下文错误、重复关闭错误能被包装返回、嵌套目录复制失败能向上传播、GitHub 临时目录创建失败不会执行 runner。

Unix 专属测试继续覆盖 FIFO 这类非普通 skeleton 条目会被拒绝，避免默认测试包在 Windows 上编译依赖 Unix-only syscall。

## 风险和优化建议

风险：`copyFile` 现在拒绝任何已存在目标条目，这会让重复执行 skeleton 复制在目标已有文件时失败。该行为符合本次加固要求，但上层如果需要幂等覆盖策略，应显式设计单独的确认流程。

优化建议：后续可以在 CLI 层为目标目录非空、已有文件冲突等场景提供更明确的用户提示。

## 孤儿 / 死代码

未发现本次变更引入的孤儿代码或死代码。

## 兼容性 / fallback 代码

未新增兼容性保留代码或 fallback 逻辑。

## 未完成 / 待办事项

无。
