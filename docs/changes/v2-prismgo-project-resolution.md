# PrismGo 项目目标解析与目录安全

## Feature overview and implementation goals

本次变更实现 `prismgo new <name>` 的项目解析层，目标是在真正创建应用前先得到可测试的项目计划，包括本地目录名、目标目录绝对路径和 Go module 路径，并阻止不安全或会覆盖数据的目标路径。

## Requirements / business background

Task 2 要求新增 `internal/project` 包，将 CLI 传入的项目名称、`--module` 和 `--force` 转换为明确的创建计划。该阶段仍不执行最终创建操作，保持后续创建流程为未实现状态。

## Impact scope

影响范围为 `prismgo new` 命令的参数解析后置校验：合法输入会继续返回 Task 1 的未实现错误；不合法项目名、路径穿越、已有目录或文件冲突会提前返回明确错误。

## Which files were modified

- `internal/project/project.go`：新增项目解析、模块推断、目标目录解析和目录安全校验。
- `internal/project/project_test.go`：新增项目解析和目录安全单元测试。
- `internal/cli/new.go`：接入 `project.Resolve`，保留 `--github` unsupported 行为和最终未实现占位错误。
- `internal/cli/new_test.go`：补充 CLI 对模块路径解析和不安全路径拒绝的覆盖。

## What behavioral changes were made

- `myapp` 解析为目录 `myapp`，模块 `myapp`。
- `github.com/acme/myapp` 解析为目录 `myapp`，模块 `github.com/acme/myapp`。
- `--module github.com/acme/service` 会覆盖模块名，但不改变目录名。
- 空项目名、路径穿越、绝对路径和空路径段会失败。
- 目标目录已存在时，默认失败；`--force` 只允许复用已存在的空目录。
- 已存在的非空目录或文件路径即使带 `--force` 也会失败。
- 不会删除任何目标路径。

## Which checks were executed and a summary of the results

- `go test ./internal/project`：先按 TDD 运行，因 `Resolve` 和 `Options` 未实现而失败，符合预期。
- `go test ./internal/project ./internal/cli`：通过。
- `go test ./...`：通过。
- `make test PACKAGES='./internal/project ./internal/cli'`：通过。
- `make test`：通过。
- `golangci-lint run --verbose ./internal/project ./internal/cli`：通过，0 issues。
- `make fmt-check`：通过。
- `./.github/scripts/coverage.sh ./internal/project ./internal/cli`：直接执行因脚本无执行权限失败。
- `bash ./.github/scripts/coverage.sh ./internal/project ./internal/cli`：通过，总语句覆盖率 `91.2%`。

## What logic is covered by unit tests

单元测试覆盖普通名称、模块路径名称、显式模块覆盖、空名称、路径穿越、绝对路径、空路径段、默认当前工作目录、已存在目录、`--force` 复用空目录、`--force` 拒绝非空目录、已存在文件冲突，以及 CLI 在接入解析器后仍保留未实现占位错误和不安全路径提前失败。

## Risks and optimization suggestions

当前解析规则按计划使用“最后一个斜杠前存在点号”判断模块路径，未额外支持 `acme/myapp` 这类无点号的模块路径语义。后续如需要支持更多 Go module 输入形式，应先明确目录推断规则。

## Orphaned/dead code

未发现由本次变更引入的孤立或死代码。

## Compatibility/fallback code

未发现由本次变更引入的兼容或 fallback 代码。

## Outstanding/incomplete items

最终项目创建流程仍按计划保留为未实现，等待后续 Task 实现。
