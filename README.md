# Jiasine CLI

Jiasine Cross-platform multi-language support system。

## 功能概要

### 核心定位

JiasineCli 是一个 **跨平台多语言统一调用系统**，用 Go 作为"胶水层"，把不同语言编写的能力模块统一管理起来。无论底层用什么语言实现，上层都通过一个 CLI 工具统一调用。

```
用户 → jiasinecli → ┬─ Bridge层 (FFI)      → 本地动态库 (.dll/.so/.dylib)
                     ├─ Service层 (HTTP)     → 独立运行的后端服务
                     └─ Plugin层 (可执行文件)  → 用户自定义扩展
```

### 三层调用说明

| 层 | 调用方式 | 支持语言 | 适用场景 |
|---|---|---|---|
| **Bridge** | FFI 直接加载 DLL/SO | C, Rust, Objective-C, .NET AOT | 高性能计算、密码学、图像处理等需要低延迟的场景 |
| **Service** | HTTP / 子进程调用 | Python, C#, JS, TS, Java, Swift | AI 推理、Web 服务、数据处理等独立服务 |
| **Plugin** | 可执行文件 / 共享库 | 任意语言 | 用户自定义扩展功能 |

### 可扩展实现的功能

**1. AI / 机器学习管道**
- Python 写 AI 推理服务 → Service 层 HTTP 调用
- 例：`jiasinecli service call ai-model --func predict --data "input.jpg"`

**2. 高性能计算模块**
- C/Rust 写计算密集库 → Bridge 层 FFI 调用
- 例：图像压缩、加密解密、音视频编解码

**3. 跨平台自动化运维**
- 编译成 7 个平台的二进制，统一运维命令
- 例：`jiasinecli service health` 批量检查服务状态

**4. 微服务编排**
- 注册多个 HTTP 服务（Java、C#、Node.js），统一入口调用
- 例：`jiasinecli service call payment --func charge --params '{"amount":100}'`

**5. 本地工具链整合**
- Rust 写网络工具 → 编译为 DLL → Bridge 层调用
- .NET AOT 编译业务逻辑 → 零依赖调用

**6. 插件生态**
- 任何人用任何语言写一个符合协议的可执行文件
- 放入 `~/.jiasine/plugins/` 即可被发现和调用

**7. 物联网 / 嵌入式**
- 已有 ARM（树莓派）和 ARM64 交叉编译
- 可在边缘设备上统一调用各语言模块

### 使用示例

```bash
# 调用 C 写的加密库
jiasinecli bridge call crypto --func encrypt --params "hello"

# 调用 Python AI 服务
jiasinecli service call ai-service --func predict --params '{"image":"a.jpg"}'

# 查看所有服务健康状态
jiasinecli service health

# 运行 9 语言集成测试
jiasinecli test --lang all

# 安装第三方插件
jiasinecli plugin install my-tool
```

### 核心优势

- **Go 作为胶水层** — 编译快、跨平台、单二进制、并发能力强
- **9 种语言支持** — 几乎覆盖所有主流编程语言
- **7 平台编译** — Windows / macOS / Linux × x64 / ARM
- **双模式交互** — 命令行模式 + 交互式 Shell（双击 exe 即用）
- **配置驱动** — 通过 YAML 注册新的库和服务，无需改代码

> 简单说：**这是一个"万能遥控器"——底层能力用最适合的语言实现，上层用一个 CLI 统一指挥。**

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
# 构建 (所有平台统一使用 Python 脚本)
python build.py          # 本地构建
python build.py cross    # 跨平台全量编译

# 查看帮助
jiasinecli --help

# 查看版本
jiasinecli version

# 运行集成测试 (需要 gcc/rust/python/dotnet/node 等)
jiasinecli test --lang all
```

## 安装到其他电脑

### Windows

**方法一：一键安装（推荐）**

1. 将 `jiasinecli-windows.exe` 复制到目标电脑任意位置
2. 双击运行（自动以管理员权限打开 PowerShell）
3. 在打开的 PowerShell 中执行：
   ```powershell
   jiasinecli-windows.exe setup
   ```
4. 重新打开一个 PowerShell 窗口，即可直接使用：
   ```powershell
   jiasinecli          # 启动 AI 模式
   jiasine             # 同上（别名）
   ```

`setup` 命令会自动完成：
- 复制程序到 `%USERPROFILE%\.jiasine\bin\jiasinecli.exe`
- 将该目录添加到用户 PATH（通过注册表，永久生效）
- 创建 `jiasine.cmd` 别名

**方法二：手动安装**

1. 将 `jiasinecli-windows.exe` 重命名为 `jiasinecli.exe`
2. 放到一个固定目录，例如 `C:\Tools\`
3. 将该目录添加到系统 PATH：
   ```powershell
   [Environment]::SetEnvironmentVariable('PATH', $env:PATH + ';C:\Tools', 'User')
   ```
4. 重新打开 PowerShell 即可使用 `jiasinecli`

### macOS

```bash
# Apple Silicon (M1/M2/M3/M4)
chmod +x jiasinecli-macos-arm
sudo cp jiasinecli-macos-arm /usr/local/bin/jiasinecli

# Intel Mac
chmod +x jiasinecli-macos-intel
sudo cp jiasinecli-macos-intel /usr/local/bin/jiasinecli
```

### Linux

```bash
# x64
chmod +x jiasinecli-linux
sudo cp jiasinecli-linux /usr/local/bin/jiasinecli

# ARM64 (如 AWS Graviton)
chmod +x jiasinecli-linux-arm64
sudo cp jiasinecli-linux-arm64 /usr/local/bin/jiasinecli

# Raspberry Pi
chmod +x jiasinecli-raspi
sudo cp jiasinecli-raspi /usr/local/bin/jiasinecli
```

安装后所有平台均可直接运行 `jiasinecli`。

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
├── ai                  # AI 大模型交互
│   ├── chat            # 与 AI 对话 (--provider, --model, --agent)
│   ├── provider        # 服务商管理
│   │   ├── list        # 列出已配置的服务商
│   │   └── switch      # 切换当前服务商
│   ├── agent           # Agent 智能体管理
│   │   ├── list        # 列出所有 Agent
│   │   └── run         # 运行指定 Agent
│   └── skill           # Skills 技能管理
│       ├── list        # 列出所有 Skill
│       ├── install     # 安装 Skill (JSON)
│       └── remove      # 卸载 Skill
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

## AI 插件

内置 AI 插件支持主流大模型统一调用，通过配置文件管理服务商和 API 密钥。

### 支持的 AI 服务商

| 服务商 | 别名 | 默认模型 | 其他模型 |
|---|---|---|---|
| **OpenAI** | openai, chatgpt | gpt-4o | gpt-4o-mini, o1, o3-mini |
| **Anthropic** | claude, anthropic | claude-sonnet-4-20250514 | claude-opus-4-20250514 |
| **Google** | gemini, google | gemini-2.5-pro | gemini-2.5-flash |
| **阿里云** | qwen, tongyi | qwen-max | qwen-plus, qwen-turbo |
| **DeepSeek** | deepseek | deepseek-chat | deepseek-coder, deepseek-reasoner |

### AI 配置

```yaml
# ~/.jiasine/config.yaml
ai:
  active: openai       # 当前激活的服务商
  providers:
    openai:
      api_key: "sk-xxxx"
      model: "gpt-4o"
      enabled: true
    claude:
      api_key: "sk-ant-xxxx"
      model: "claude-sonnet-4-20250514"
      enabled: true
    deepseek:
      api_key: "sk-xxxx"
      model: "deepseek-chat"
      enabled: true
```

### 使用示例

```bash
# 与默认 AI 对话
jiasinecli ai chat "解释 Go 的 interface 机制"

# 指定服务商和模型
jiasinecli ai chat -p claude -m claude-opus-4-20250514 "编写一个排序函数"

# 查看已配置的服务商
jiasinecli ai provider list

# 切换当前服务商
jiasinecli ai provider switch deepseek
```

### Agent 智能体

Agent 是预配置的 AI 助手，包含特定的系统提示词和技能组合。

| 内置 Agent | 描述 |
|---|---|
| **assistant** | 通用 AI 助手 — 回答问题、写作、翻译 |
| **coder** | 编程助手 — 代码生成、调试、重构 |
| **translator** | 翻译助手 — 多语言互译 |
| **devops** | 运维助手 — 部署、监控、故障排查 |

```bash
# 查看所有 Agent
jiasinecli ai agent list

# 使用编程助手
jiasinecli ai agent run coder "帮我写一个 HTTP 中间件"

# 通过 chat 指定 Agent
jiasinecli ai chat -a translator "翻译: Hello World"
```

自定义 Agent 放入 `~/.jiasine/agents/` 目录，JSON 格式：

```json
{
  "name": "my-agent",
  "description": "我的自定义智能体",
  "system": "你是一个...",
  "skills": ["code-review", "doc-writer"],
  "temperature": 0.7
}
```

### Skills 技能系统

Skill 是可组合的能力模块，可挂载到 Agent 上增强其专业能力。

| 内置 Skill | 描述 | 标签 |
|---|---|---|
| **code-review** | 代码审查 — 质量、安全、性能分析 | code, review, quality |
| **sql-expert** | SQL 专家 — 编写、优化、调试 | sql, database |
| **api-designer** | API 设计 — RESTful/GraphQL/gRPC | api, rest, design |
| **git-helper** | Git 助手 — 分支、冲突、工作流 | git, vcs |
| **doc-writer** | 文档写手 — 技术文档、README | documentation, writing |

```bash
# 查看所有 Skill
jiasinecli ai skill list

# 安装自定义 Skill
jiasinecli ai skill install my-skill.json

# 卸载 Skill
jiasinecli ai skill remove my-skill
```

## 跨平台构建

支持 **7 个目标平台**，CGO_ENABLED=0 纯静态编译，单文件无依赖分发：

### 统一使用 Python 构建脚本

```bash
# 本地构建（自动识别当前平台）
python build.py
python build.py local

# 跨平台全量编译（编译所有 7 个平台）
python build.py cross

# 清理构建产物
python build.py clean
```

**为什么使用 Python 脚本？**
- ✅ 跨平台支持：Windows / Linux / macOS 统一使用
- ✅ 无需额外工具：只需 Python 3.6+ 和 Go
- ✅ 彩色输出：编译进度和结果清晰可见
- ✅ 自动统计：文件大小、编译成功率

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
│   ├── plugin.go                    # 插件命令
│   └── ai.go                        # AI 插件命令 (chat/provider/agent/skill)
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
│   ├── ai/                          # AI 插件
│   │   ├── provider.go              # Provider 接口 & 工厂注册
│   │   ├── providers.go             # 5 大服务商实现
│   │   ├── manager.go               # AI 统一管理器
│   │   ├── agent.go                 # Agent 智能体框架
│   │   └── skill.go                 # Skills 技能系统
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
├── build.py                         # 跨平台构建脚本 (7 targets)
├── go.mod
└── go.sum
```

## 技术栈

- **CLI 框架**: [Cobra](https://github.com/spf13/cobra) + [Pflag](https://github.com/spf13/pflag)
- **配置管理**: [Viper](https://github.com/spf13/viper) (YAML + 环境变量)
- **日志**: [Zap](https://go.uber.org/zap) (结构化日志)
- **FFI**: syscall (Windows DLL) / cgo (Unix .so/.dylib)
- **构建**: Go 原生交叉编译，CGO_ENABLED=0 纯静态链接
- **AI 插件**: 5 大主流模型服务商 + Agent 智能体 + Skills 技能系统
- **支持语言**: C, Python, Rust, C#, JavaScript, TypeScript, Java, Swift, Objective-C
