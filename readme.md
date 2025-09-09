# gocli

[![Go Version](https://img.shields.io/badge/go-1.25.0+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

gocli 是一个强大的命令行工具，用于高效管理 Go 项目。它提供了丰富的功能，包括项目初始化、构建、测试、依赖管理、代码格式化和文档生成等，帮助开发者更便捷地处理 Go 项目的日常开发任务。

## 功能特性

- 🚀 **项目管理**: 初始化、构建、运行和测试 Go 项目
- 📦 **依赖管理**: 添加、更新和分析项目依赖
- 🔧 **代码质量**: 代码检查、格式化和文档生成
- 🛠️ **工具集成**: 支持多种开发工具和配置
- 🎨 **美观输出**: 支持多种输出格式和样式
- ⚡ **高性能**: 并发处理和优化的执行效率

## 安装

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/yeisme/gocli.git
cd gocli

# 构建项目
go build -o gocli ./cmd/gocli

# 安装到系统路径（可选）
go install ./cmd/gocli
```

### 使用 Go Install

```bash
go install github.com/yeisme/gocli/cmd/gocli@latest
```

## 快速开始

### 初始化新项目

```bash
# 在当前目录初始化项目
gocli project init .

# 创建新项目目录
gocli project init myproject

# 使用模板初始化项目
gocli project init myweb --template basic
```

### 构建和运行

```bash
# 构建项目
gocli project build

# 运行项目
gocli project run

# 启用热重载运行
gocli project run --hot-reload
```

### 测试和代码质量

```bash
# 运行测试
gocli project test

# 代码检查
gocli project lint

# 格式化代码
gocli project fmt
```

## 命令列表

### 项目管理

| 命令                  | 描述             |
| --------------------- | ---------------- |
| `gocli project init`  | 初始化新 Go 项目 |
| `gocli project build` | 构建 Go 项目     |
| `gocli project run`   | 运行 Go 项目     |
| `gocli project test`  | 运行项目测试     |
| `gocli project list`  | 列出项目包       |
| `gocli project info`  | 显示项目信息     |

### 依赖管理

| 命令                   | 描述         |
| ---------------------- | ------------ |
| `gocli project add`    | 添加项目依赖 |
| `gocli project update` | 更新项目依赖 |
| `gocli project deps`   | 管理项目依赖 |

### 代码质量

| 命令                 | 描述       |
| -------------------- | ---------- |
| `gocli project lint` | 代码检查   |
| `gocli project fmt`  | 代码格式化 |
| `gocli project doc`  | 生成文档   |

### 工具管理

| 命令                  | 描述         |
| --------------------- | ------------ |
| `gocli tools install` | 安装开发工具 |
| `gocli tools list`    | 列出可用工具 |
| `gocli tools update`  | 更新工具     |

### 配置管理

| 命令                | 描述         |
| ------------------- | ------------ |
| `gocli config init` | 初始化配置   |
| `gocli config show` | 显示当前配置 |

## 使用示例

### 项目初始化

```bash
# 初始化包含多种工具配置的项目
gocli project init myapp --git --go-task --goreleaser --docker

# 使用特定模板
gocli project init webapp --template basic --license MIT
```

### 构建选项

```bash
# 发布模式构建
gocli project build --release-mode -o bin/myapp

# 调试模式构建
gocli project build --debug-mode

# 启用竞争检测
gocli project build --race
```

### 依赖分析

```bash
# 查看依赖树
gocli project deps --tree

# 检查依赖更新
gocli project deps --update --json

# 整理依赖
gocli project deps --tidy
```

### 文档生成

```bash
# 生成包文档
gocli project doc ./pkg/utils

# 生成 Markdown 格式文档
gocli project doc ./pkg/utils --style markdown --output docs/utils.md
```

## 配置

gocli 支持多种配置方式：

- 全局配置文件：`~/.gocli/config.yaml`
- 项目配置文件：`./.gocli.yaml`
- 环境变量
- 命令行参数

### 示例配置文件

```yaml
app:
  verbose: false
  quiet: false
  debug: false

project:
  default_template: basic
  auto_git_init: true
  auto_go_mod_init: true

tools:
  golangci_lint:
    enabled: true
    config_path: .golangci.yml
```

## 开发

### 构建要求

- Go 1.25.0+
- 支持的操作系统：Linux, macOS, Windows

### 开发设置

```bash
# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建所有组件
go build ./cmd/gocli
go build ./cmd/gox
go build ./cmd/schema
```

### 项目结构

```txt
gocli/
├── cmd/           # 命令行入口
├── pkg/           # 核心包
│   ├── configs/   # 配置管理
│   ├── context/   # 上下文管理
│   ├── models/    # 数据模型
│   ├── project/   # 项目管理
│   ├── style/     # 输出样式
│   ├── tools/     # 工具管理
│   └── utils/     # 工具函数
├── docs/          # 文档
├── test/          # 测试文件
└── tmp/           # 临时文件
```

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 开发规范

- 遵循 Go 编码规范
- 添加单元测试
- 更新相关文档
- 确保所有测试通过

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

感谢所有为 gocli 项目做出贡献的开发者！

特别感谢以下开源项目：

- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown 渲染
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - 样式库
