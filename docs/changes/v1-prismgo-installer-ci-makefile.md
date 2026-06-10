# v1 PrismGo Installer CI 与 Makefile 适配

## Feature overview and implementation goals

本次变更将 `.github` 与 `Makefile` 从 Prismgo Lens / 通用 Go 模板调整为 PrismGo Installer 项目配置。目标是在当前尚未添加 `go.mod`、Go 源码和 `cmd/prismgo` 的阶段保持 CI 绿色，同时明确未来安装器 CLI 的构建、安装和冒烟验证入口。

## Requirements / business background

PrismGo Installer 的定位类似 Laravel 的 `laravel new {yourapp}`，未来用户将通过 `prismgo new tmp/app` 创建 PrismGo 应用。本阶段只适配仓库工程配置，不实现安装器命令，也不验证生成项目。

## Impact scope

影响范围包括 Makefile 本地命令、GitHub Actions CI、CodeQL 工作流、Dependabot 配置、Pull Request 模板和 Issue 模板。不会修改生产代码、公共 API 或生成项目行为。

## Which files were modified

- `Makefile`
- `.github/workflows/ci.yml`
- `.github/workflows/codeql.yml`
- `.github/dependabot.yml`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/ISSUE_TEMPLATE/bug-report.yaml`
- `.github/ISSUE_TEMPLATE/feature-request.yaml`
- `.github/ISSUE_TEMPLATE/config.yml`
- `docs/changes/v1-prismgo-installer-ci-makefile.md`

## What behavioral changes were made

`make ci` 会在没有 `go.mod` 时跳过 Go 检查并成功退出。`make build` 和 `make install` 会在没有 `cmd/prismgo` 时跳过并成功退出。`make smoke-new` 暂时只输出延后执行的提示，不会运行 `prismgo new tmp/app`。GitHub Actions 通过 `make ci` 执行当前可用检查。

## Which checks were executed and a summary of the results

执行 `make ci`、`make build`、`make install`、`make smoke-new`。由于当前仓库尚未包含 `go.mod` 和 `cmd/prismgo`，这些命令应输出跳过提示并以成功状态结束。执行 `.github` 文案搜索，确认不再包含 Prismgo Lens 专属引用。

## What logic is covered by unit tests

本次变更不包含 Go 源码和单元测试。覆盖的逻辑是 Makefile 的仓库状态判断：缺少 `go.mod` 时跳过 Go 检查，缺少 `cmd/prismgo` 时跳过 CLI 构建与安装，生成项目冒烟验证暂不执行。

## Risks and optimization suggestions

跳过逻辑如果长期保留，可能掩盖安装器实现缺失。建议在添加 `go.mod` 和 `cmd/prismgo` 后，重新评估是否让 CI 强制执行构建、测试和 `prismgo new tmp/app` 冒烟验证。

## Orphaned/dead code

本次变更未新增孤立代码。旧的 Prismgo Lens 专属配置文案会被替换，不保留无用途的 Lens 字段。

## Compatibility/fallback code

本次变更包含临时跳过逻辑，用于适配当前没有 Go 模块和 CLI 源码的仓库状态。该逻辑不是面向生产行为的兼容代码，后续源码补齐后应重新确认是否继续保留。

## Outstanding/incomplete items

尚未实现 `prismgo` CLI。尚未添加 `go.mod`。尚未启用 `prismgo new tmp/app` 生成项目冒烟验证。尚未恢复 Dependabot 的 `gomod` 更新。
