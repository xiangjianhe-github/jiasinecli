# Jiasine CLI

Jiasine Cross-platform multi-language support system。

## 架构

```
┌──────────────────────────────────────────────────────┐
│                   CLI 层 (Go/Cobra)                   │
│           命令解析 · 并发控制 · 用户体验               │
├────────────┬────────────────┬────────────────────────┤
│  插件层     │   桥接层 (FFI)  │     服务层 (RPC)       │
│  Plugin    │   Bridge       │     Service            │
│            │                │                        │
│  可执行文件  │  C 动态库       │  HTTP 服务 (Python)    │
│  共享库     │  Rust 动态库    │  HTTP 服务 (C#)        │
│            │  Obj-C 动态库   │  HTTP 服务 (JS/TS)     │
│            │  .NET AOT DLL  │  HTTP 服务 (Java)      │
│            │                │  子进程调用             │
│            │                │  Swift 编译执行         │
└────────────┴────────────────┴────────────────────────┘
```

**三层调用方式：**

| 层 | 说明 | 适用场景 |
|---|---|---|
| **Bridge** | FFI 调用动态库 (syscall/dlopen) | C、Rust、Objective-C、.NET Native AOT 编译的 .dll/.so/.dylib |
| **Service** | HTTP/进程调用 | Python、C#、JavaScript、TypeScript、Java、Swift 独立运行服务 |
| **Plugin** | 可执行文件插件 | 用户扩展功能 |

## 快速开始

```bash
# 构建 (Windows)
.\build.ps1 -Target dev

# 构建 (Linux/macOS)
make dev

# 查看帮助
jiasinecli --help

# 查看版本
jiasinecli version

# 运行集成测试 (需要 gcc/rust/python/dotnet/node 等)
jiasinecli test --lang all
```

## 命令一览

```
jiasinecli
├── version             # 版本信息 (SemVer 2.0)
│   └── --short         # 仅显示版本号
├── test                # 集成测试
│   ├── --lang c        # 仅测试 C
│   ├── --lang python   # 仅测试 Python
│   ├── --lang rust     # 仅测试 Rust
│   ├── --lang csharp   # 仅测试 C#
│   ├── --lang js       # 仅测试 JavaScript
│   ├── --lang typescript # 仅测试 TypeScript
│   ├── --lang java     # 仅测试 Java
│   ├── --lang swift    # 仅测试 Swift
│   ├── --lang objc     # 仅测试 Objective-C
│   ├── --lang all      # 测试所有语言
│   └── status          # 查看工具链就绪状态
├── bridge              # 桥接层 (FFI 动态库)
│   ├── list            # 列出已加载的动态库
│   └── call            # 调用动态库函数
├── service             # 服务管理
│   ├── list            # 列出已注册的服务
│   ├── call            # 调用远程服务
│   └── health          # 健康检查
├── plugin              # 插件管理
│   ├── list            # 列出已安装插件
│   ├── install         # 安装插件
│   └── remove          # 卸载插件
└── completion          # Shell 自动补全
```

## 版本控制规则

遵循 [SemVer 2.0](https://semver.org/) 规范：

```
MAJOR.MINOR.PATCH[-prerelease][+buildmetadata]
```

| 字段 | 说明 | 升版条件 |
|---|---|---|
| MAJOR | 主版本号 | 含有不兼容的 API 变更 |
| MINOR | 次版本号 | 向下兼容的功能新增 |
| PATCH | 补丁号 | 向下兼容的问题修正 |
| prerelease | 预发布标识 | alpha → beta → rc 递进 |
| buildmetadata | 构建元数据 | CI commit hash / build date |

版本兼容性判定：
- MAJOR 相同即兼容
- MAJOR=0 时为开发阶段，MINOR 变更也可能不兼容

## 集成测试

项目包含 9 种语言的测试用例，验证 Go 胶水层对各语言资产的调用能力：

| 语言 | 调用方式 | 测试项 | 源码 |
|---|---|---|---|
| **C** | FFI (DLL/SO) | add, get_version, reverse_string, health | `tests/c/` |
| **Python** | HTTP + 子进程 | health, version, add, reverse, fibonacci, upper | `tests/python/` |
| **Rust** | FFI (cdylib) | add, get_version, reverse_string, hash, health | `tests/rust/` |
| **C#** | HTTP (ASP.NET) | health, version, add, reverse, factorial | `tests/csharp/` |
| **JavaScript** | HTTP + 子进程 | health, version, add, reverse, fibonacci, factorial | `tests/js/` |
| **TypeScript** | HTTP + 子进程 | health, version, add, reverse, fibonacci, factorial | `tests/typescript/` |
| **Java** | HTTP + 子进程 | health, version, add, reverse, fibonacci, factorial | `tests/java/` |
| **Swift** | 编译 + 子进程 + HTTP | health, version, add, reverse, fibonacci, factorial | `tests/swift/` |
| **Objective-C** | FFI (DLL/SO) | add, get_version, reverse_string, health | `tests/objc/` |

```bash
# 查看工具链状态
jiasinecli test status

# 运行全部测试 (28 项)
jiasinecli test --lang all

# 仅测试单一语言
jiasinecli test --lang rust
```

## 配置

配置文件位于 `~/.jiasine/config.yaml`，参见 [config.example.yaml](config.example.yaml)。

```yaml
# 注册一个 Python HTTP 服务
services:
  ai-service:
    type: http
    address: "http://localhost:8001"
    health_check: "/health"
    description: "AI 推理服务"

# 注册一个 Rust 动态库
bridges:
  crypto-lib:
    type: rust
    platform:
      windows: "libs/jiasine_crypto.dll"
      linux: "libs/libjiasine_crypto.so"
      darwin: "libs/libjiasine_crypto.dylib"
    functions: ["encrypt", "decrypt", "hash"]
```

## 跨平台构建

支持 **7 个目标平台**，CGO_ENABLED=0 纯静态编译，单文件无依赖分发：

```powershell
# PowerShell - 编译所有平台
.\build.ps1 -Target cross

# PowerShell - 仅编译指定平台
.\build.ps1 -Target windows
.\build.ps1 -Target linux
.\build.ps1 -Target darwin
.\build.ps1 -Target raspi
```

```bash
# Make - 编译所有平台
make cross

# Make - 仅编译指定平台
make cross-windows
make cross-linux
make cross-darwin
make cross-raspi
```

编译产物在 `dist/` 目录：

| 平台 | GOOS/GOARCH | 文件 |
|---|---|---|
| Windows x64 | windows/amd64 | `jiasinecli.exe` |
| Windows ARM64 | windows/arm64 | `jiasinecli-windows-arm64.exe` |
| macOS Intel | darwin/amd64 | `jiasinecli-macos-intel` |
| macOS Apple Silicon | darwin/arm64 | `jiasinecli-macos-arm` |
| Linux x64 | linux/amd64 | `jiasinecli-linux` |
| Linux ARM64 | linux/arm64 | `jiasinecli-linux-arm64` |
| Raspberry Pi | linux/arm (GOARM=7) | `jiasinecli-raspi` |

## 项目结构

```
JiasineCli/
├── main.go                          # 入口
├── cmd/                             # CLI 命令定义
│   ├── root.go                      # 根命令 & 初始化
│   ├── version.go                   # 版本命令 (SemVer 2.0)
│   ├── test.go                      # 集成测试命令
│   ├── bridge.go                    # 桥接层命令
│   ├── service.go                   # 服务层命令
│   └── plugin.go                    # 插件命令
├── internal/                        # 内部实现
│   ├── version/                     # 版本管理
│   │   ├── version.go               # SemVer 解析/比较
│   │   └── version_test.go          # 版本单元测试
│   ├── testrunner/                  # 集成测试运行器
│   │   ├── runner.go                # 多语言测试调度
│   │   ├── ffi_test_windows.go      # Windows DLL 测试
│   │   └── ffi_test_other.go        # Unix 平台 stub
│   ├── bridge/                      # FFI 桥接层
│   │   ├── manager.go               # 桥接管理器
│   │   ├── ffi.go                   # FFI 通用逻辑
│   │   ├── ffi_windows.go           # Windows DLL 调用
│   │   └── ffi_other.go             # Unix 平台 stub
│   ├── service/                     # 远程服务调用
│   │   └── manager.go               # 服务管理器 (HTTP/gRPC/Process)
│   ├── plugin/                      # 插件系统
│   │   └── manager.go               # 插件管理器
│   ├── config/                      # 配置管理
│   │   └── config.go                # Viper 配置加载
│   └── logger/                      # 日志管理
│       └── logger.go                # Zap 结构化日志
├── tests/                           # 多语言测试资产
│   ├── c/                           # C 共享库测试
│   ├── python/                      # Python HTTP/进程 测试
│   ├── rust/                        # Rust cdylib 测试
│   ├── csharp/                      # C# ASP.NET 测试
│   ├── js/                          # JavaScript HTTP/进程 测试
│   ├── typescript/                  # TypeScript HTTP/进程 测试
│   ├── java/                        # Java HTTP/进程 测试
│   ├── swift/                       # Swift 编译/HTTP/进程 测试
│   └── objc/                        # Objective-C 共享库测试
├── config.example.yaml              # 配置示例
├── build.ps1                        # Windows 构建脚本 (7 targets)
├── Makefile                         # Linux/macOS 构建脚本 (7 targets)
├── go.mod
└── go.sum
```

## 技术栈

- **CLI 框架**: [Cobra](https://github.com/spf13/cobra) + [Pflag](https://github.com/spf13/pflag)
- **配置管理**: [Viper](https://github.com/spf13/viper) (YAML + 环境变量)
- **日志**: [Zap](https://go.uber.org/zap) (结构化日志)
- **FFI**: syscall (Windows DLL) / cgo (Unix .so/.dylib)
- **构建**: Go 原生交叉编译，CGO_ENABLED=0 纯静态链接
- **支持语言**: C, Python, Rust, C#, JavaScript, TypeScript, Java, Swift, Objective-C
