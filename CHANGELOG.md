# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed (2026-03-06 晚间修复)
- 🐛 **彻底修复 PowerShell 蓝色背景问题**: 使用 Windows API 填充整个屏幕缓冲区为黑色背景
  - 方法1: SetConsoleTextAttribute API 设置默认属性
  - 方法2: FillConsoleOutputAttribute 填充整个屏幕
  - 方法3: ANSI 转义序列作为额外保障
  - 在所有输出前强制添加 `ForceBlackBg` 前缀
- 🐛 **修复跨平台编译失败**: 为非 Windows 平台添加 `shell_unix.go` 空实现
- 🚀 **新增 Python 构建脚本**: `build.py` 支持在所有平台上编译
  - 替代 PowerShell 脚本（`build.ps1` 仅 Windows 可用）
  - 支持 Windows/Linux/macOS 全平台编译
  - 彩色终端输出，编译进度显示
  - 自动统计文件大小

### 规划中的功能
- 多 Agent 分工协作系统
- 向量数据库集成（长期记忆）
- 自主学习和技能进化
- Skill 市场和企业级功能
- 多模态能力（视觉、语音、视频）
- 智能化开发工作流
- 代码健康度监控
- 沙箱安全系统
- 历史对话交互式浏览（方向键导航）

详见 [LONG_TERM_ROADMAP.md](./docs/LONG_TERM_ROADMAP.md)

## [0.2.0-alpha] - 2026-03-06

### 🚀 重大更新：历史记录功能

#### Added
- ✨ **SQLite 历史记录存储**: 所有 AI 对话自动持久化保存
  - 会话管理：自动创建和结束会话
  - 消息保存：用户输入和 AI 回复完整记录
  - 元数据追踪：时间戳、token 使用量、提供商信息
- 📚 **历史记录命令行接口**:
  - `history sessions` - 列出所有历史会话（支持筛选和分页）
  - `history show <session-id>` - 查看特定会话的完整对话内容
  - `history search <keyword>` - 关键词搜索历史消息
  - `history delete <session-id>` - 删除指定会话
  - `history clear --before <date>` - 清理旧会话
  - `history stats` - 查看统计信息（会话数、消息数等）
- 🎨 **美化的历史展示**:
  - 彩色标记：会话信息、角色区分（用户/AI）
  - Markdown 渲染：历史消息支持完整的格式化显示
  - 分隔线和图标：清晰的视觉层次
- 🔧 **纯 Go 实现**: 使用 modernc.org/sqlite（无需 CGO）
  - 跨平台兼容：支持所有目标平台
  - 简化部署：单个可执行文件包含所有依赖

#### Changed
- 🔗 **AI 对话集成**: `ai chat` 自动保存到历史数据库
  - 每轮对话实时保存用户消息和 AI 回复
  - 会话元数据：Agent、提供商、模型、开始/结束时间
  - Token 统计：自动记录每次 AI 回复的 token 使用量

#### Technical Details
- 新增包: `internal/history` (417 行核心代码)
- 新增命令: `cmd/history.go` (440+ 行 CLI 接口)
- 数据库位置: `~/.jiasine/history.db`
- 数据库架构:
  - `sessions` 表：会话元数据（ID、Agent、提供商、模型、时间、消息数、标签）
  - `messages` 表：消息内容（ID、会话 ID、角色、内容、时间、tokens、元数据）
  - 索引优化：session_id 和 timestamp 建立索引提升查询性能
- 单元测试: 8 个测试函数，100% 通过

#### Fixed
- 🐛 修复 Markdown 渲染中 ANSI 重置导致的背景色丢失问题
- 🎨 优化彩色标题的显示效果（使用柔和的灰色分隔线）

### 文档更新
- 📖 工作完成总结（本次会话）
- 📝 历史功能设计文档已实现 Phase 1（基础存储）和 Phase 2（CLI 命令）

## [0.1.1-alpha] - 2026-03-06

### Added
- ✨ **PowerShell Markdown 渲染优化**:
  - 彩色标题：# 绿色、## 黄色、### 紫色、#### 蓝色（使用柔和的 256 色调色板）
  - 标题装饰：不同符号标记（═══ / ▸ / • / ›）区分层级
  - 代码块边框：使用 Unicode 线条字符（┌─ ─┐ │ └─）绘制边框
  - 语法高亮：代码块内的关键字、函数名、字符串等颜色区分
  - 背景保持：ANSI 重置后自动恢复黑色背景 + 白色文字
- ✨ **PowerShell 背景色优化**: 双击启动时自动将控制台背景色设置为黑色（原来是 PowerShell 默认的蓝色）
- 📝 完整的中长期功能规划文档 [LONG_TERM_ROADMAP.md](./docs/LONG_TERM_ROADMAP.md)
- 📝 历史对话查看功能设计文档 [HISTORY_FEATURE_DESIGN.md](./docs/HISTORY_FEATURE_DESIGN.md)
- 📝 功能更新说明文档 [FEATURE_UPDATES.md](./docs/FEATURE_UPDATES.md)

### Changed
- 🎨 优化 `enableVirtualTerminal()` 函数，添加 ANSI 转义序列设置背景色
- 🎨 启动时清屏并设置黑色背景 + 亮白色前景
- 🎨 修改 `ansiReset` 常量保留黑色背景（`\033[0m\033[40m\033[97m`）
- 🎨 更新 Markdown 渲染器支持彩色标题和边框代码块

### Technical Details
- 修改文件:
  - `internal/shell/shell_windows.go` - 启动背景色设置
  - `internal/render/markdown.go` - Markdown 渲染优化
  - `internal/render/highlight.go` - ANSI 重置序列修复
- ANSI 码优化:
  - 背景: `\033[40m` (黑色) + `\033[97m` (亮白文字)
  - 标题颜色: `38;5;120` (绿), `38;5;228` (黄), `38;5;213` (紫), `38;5;75` (蓝)
  - 代码背景: `48;5;235` (深灰)

## [0.1.0-alpha.1] - 2026-03-05

### Added
- 🎨 **AI 对话界面优化**: VS Code/Copilot CLI 风格的暗色主题
  - Markdown 粗体支持 (`**文本**`)
  - Markdown 斜体支持 (`*文本*`, `_文本_`)
  - 优化的行内代码渲染（浅橙色文字 + 深灰背景）
  - VS Code 暗色主题配色（256 色调色板）
  - 美化的 AI 回复分隔线和元信息展示
- 🐛 **双击闪退修复**: 取消 UAC 提示时不再闪退，显示警告后继续运行
- 🔄 **版本管理系统**: SemVer 2.0 规范 + 自动更新功能
- 📦 **跨平台编译**: 支持 7 个平台 (Windows/Linux/macOS × amd64/arm64/arm)
- 🚀 **Windows Terminal 启动**: 双击时以管理员权限在 PowerShell 中启动

### Changed
- 🎨 `internal/render/markdown.go`: 新增粗体、斜体渲染，优化色彩方案
- 🎨 `internal/banner/banner.go`: 升级为 256 色调色板
- 🎨 `cmd/ai.go`: 优化 AI 回复显示样式
- 🔧 `internal/shell/shell_windows.go`: 修复 UAC 取消导致的程序退出
- 🔧 `main.go`: 条件性退出逻辑（仅成功重启时退出）

### Fixed
- 🐛 修复双击启动时取消 UAC 导致程序闪退的问题
- 🐛 修复跨平台编译缺失函数的问题（`shell_other.go`）
- 🐛 修复 Markdown 渲染中 `**` 被错误解析为两个 `*` 的问题

### Documentation
- 📝 添加 [AI_INTERFACE_OPTIMIZATION.md](./docs/AI_INTERFACE_OPTIMIZATION.md)
- 📝 添加 [MARKDOWN_QUICK_REFERENCE.md](./docs/MARKDOWN_QUICK_REFERENCE.md)
- 📝 添加 [DOUBLE_CLICK_FIX.md](./docs/DOUBLE_CLICK_FIX.md)

### Testing
- ✅ 所有 7 个平台编译成功
- ✅ Markdown 渲染单元测试通过
- ✅ 基本功能测试（version, help, update）通过

## [0.1.0-alpha] - 2026-03-01

### Added
- 🎉 初始版本发布
- 🤖 AI 对话功能 (Claude API 集成)
- 🔧 插件系统和桥接层
- 📦 多语言支持 (Go, Python, Rust, C#, JavaScript, TypeScript, Java, Swift, Objective-C)
- ⚡ 交互式 Shell 模式
- 📝 Markdown 渲染（基础支持）
- 🔄 版本检查和更新提示

---

## 版本号说明

遵循 [Semantic Versioning 2.0.0](https://semver.org/)：

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复
- **PreRelease 标签**:
  - `alpha`: 开发阶段，功能不完整
  - `beta`: 功能完整，测试阶段
  - `rc`: Release Candidate，候选发布版本

## 链接

- [GitHub Repository](https://github.com/xiangjianhe-github/jiasinecli)
- [Issue Tracker](https://github.com/xiangjianhe-github/jiasinecli/issues)
- [Documentation](./docs/)

---

**Note**: Alpha 版本可能包含未完成的功能和已知问题。不建议在生产环境使用。
