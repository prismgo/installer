# v2 恢复 CI 覆盖率矩阵

## Feature overview and implementation goals

本次变更恢复 PrismGo Installer CI 中的覆盖率数据生成流程。目标是在保留当前安装器骨架跳过逻辑的前提下，让 GitHub Actions 继续在 Ubuntu 和 macOS 上执行 `make covdata`。

## Requirements / business background

项目需要保留跨系统覆盖率验证入口。此前 CI 简化为只运行 `make ci`，遗漏了原有的 Ubuntu / macOS `make covdata` 执行路径。本次只恢复 `covdata`，不添加 race 检查，也不启用 `prismgo new tmp/app` 生成项目验证。

## Impact scope

影响范围包括 `Makefile` 的 `covdata` 目标和 `.github/workflows/ci.yml` 的 CI job 编排。不会修改生产代码、公共 API 或安装器生成行为。

## Which files were modified

- `Makefile`
- `.github/workflows/ci.yml`
- `docs/changes/v2-restore-ci-covdata-matrix.md`

## What behavioral changes were made

`Makefile` 重新提供 `make covdata` 目标。当前没有 `go.mod` 时，该目标输出跳过提示并成功退出；未来添加 Go 模块后会执行 `./.github/scripts/coverage.sh $(PACKAGES)`。CI 新增 `coverage` job，在 `ubuntu-latest` 和 `macos-latest` 上执行 `make covdata`。

## Which checks were executed and a summary of the results

执行 `make ci`、`make covdata`、`make build`、`make install`、`make smoke-new`。当前仓库没有 `go.mod` 和 `cmd/prismgo`，相关命令输出跳过提示并成功结束。执行 CI 文案搜索，确认没有恢复 Prismgo Lens 或 `gomod` Dependabot 遗留配置。

## What logic is covered by unit tests

本次变更不包含 Go 源码和单元测试。覆盖的逻辑是 Makefile 的仓库状态判断：缺少 `go.mod` 时 `make covdata` 跳过；未来存在 `go.mod` 时调用项目覆盖率脚本。

## Risks and optimization suggestions

当前 `make covdata` 不会生成覆盖率产物，因为仓库还没有 Go 模块。未来添加 `go.mod` 后，应确认 `.github/scripts/coverage.sh` 能在 Ubuntu 和 macOS 上产生 `.coverage/coverage.out`，并根据生成项目验证需求决定是否把 `smoke-new` 纳入 CI。

## Orphaned/dead code

本次变更未新增孤立代码。

## Compatibility/fallback code

本次变更保留安装器骨架阶段的跳过逻辑。该逻辑只用于当前仓库尚未添加 Go 模块时保持 CI 可运行，不影响生产行为。

## Outstanding/incomplete items

尚未实现 `prismgo` CLI。尚未添加 `go.mod`。尚未启用 `prismgo new tmp/app` 生成项目冒烟验证。当前覆盖率命令因没有 Go 包而不会产生实际 statement coverage。
