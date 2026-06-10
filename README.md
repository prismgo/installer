# PrismGo 安装器

PrismGo installer 提供 Laravel 风格的项目创建体验：

```bash
prismgo new myapp
```

它会从官方应用骨架 `github.com/prismgo/prismgo` 创建项目，重写 Go module 与项目内部 import，初始化 `.env`，并默认运行 `go mod tidy` 和 `go test ./...`。

## 安装

前置要求：

- Go 工具链，用于安装 installer，并在默认创建流程中运行 `go mod tidy` 和 `go test ./...`
- Git，用于拉取 `github.com/prismgo/prismgo` 应用骨架；即使不使用 `--git`，创建项目也需要 Git
- 能访问 GitHub 的网络环境

使用 Go 工具链安装：

```bash
go install github.com/prismgo/installer/cmd/prismgo@latest
```

或使用安装脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/prismgo/installer/main/scripts/install.sh | sh
```

确保 `go env GOPATH` 下的 `bin` 目录，或 `go env GOBIN`，已经加入 `PATH`。

## 使用

创建短名称项目：

```bash
prismgo new myapp
```

生成目录 `myapp`，`go.mod` 使用：

```go
module myapp
```

创建完整 Go module 路径项目：

```bash
prismgo new github.com/acme/myapp
```

生成目录 `myapp`，`go.mod` 使用：

```go
module github.com/acme/myapp
```

显式指定 module：

```bash
prismgo new myapp --module github.com/acme/service
```

跳过依赖安装和测试：

```bash
prismgo new myapp --no-install
```

`--no-install` 只跳过 `go mod tidy` 和 `go test ./...`。它不会跳过 skeleton 获取，仍然需要 Git 和 GitHub 访问。

初始化本地 Git 仓库：

```bash
prismgo new myapp --git
prismgo new myapp --git --branch develop
```

复用已存在的空目录：

```bash
mkdir myapp
prismgo new myapp --force
```

`--force` 只允许使用空目录，不会删除或覆盖非空目录。

## MVP 范围

当前版本聚焦核心创建流程：

- 远程获取 `github.com/prismgo/prismgo` 应用骨架
- 重写 `go.mod` module
- 重写项目内部 Go import，例如 `prismgo/bootstrap`
- 复制 `.env.example` 为 `.env`
- 默认运行 `go mod tidy` 和 `go test ./...`
- 可选本地 Git 初始化

暂不包含：

- database 预设
- starter kit / auth scaffold
- GitHub 仓库创建
- Node / 前端构建集成
- AI helper 安装
- 离线 skeleton 缓存或内置 skeleton

## 开发

运行测试：

```bash
go test ./...
```

构建本地二进制：

```bash
go build -o bin/prismgo ./cmd/prismgo
```
