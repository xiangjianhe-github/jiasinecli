# 功能更新说明 - v0.1.1

## ✅ 立即可用的改进

### 1. PowerShell 背景色优化

**问题**: 双击打开 `jiasinecli-windows.exe` 时，PowerShell 默认的蓝色背景不够美观。

**解决方案**: 启动时自动将控制台背景色设置为黑色，前景色设置为亮白色。

**技术实现**:
- 在 `shell_windows.go` 的 `enableVirtualTerminal()` 函数中添加 ANSI 转义序列
- 使用 `\033[40m` (黑色背景) + `\033[97m` (亮白色文字) + `\033[2J\033[H` (清屏)

**测试方法**:
```powershell
# 方法1: 双击 jiasinecli.exe
# 方法2: 在 PowerShell 中运行
.\jiasinecli.exe

# 应该看到黑色背景，而不是蓝色背景
```

**修改文件**: [internal/shell/shell_windows.go](../internal/shell/shell_windows.go#L30-L50)

---

## 📋 中长期规划

详细的功能规划已记录在 [LONG_TERM_ROADMAP.md](./LONG_TERM_ROADMAP.md)，包括：

### 2. 多 Agent 分工协作 (v0.2.0 - v0.5.0)
- 任务自动分解
- 专业 Agent 协同 (Code, Test, Review, Debug, Security)
- DAG 任务调度

### 3. 长期记忆与上下文管理 (v0.2.0 - v0.5.0)
- **历史对话查看**: 使用 ↑/↓ 键滚动查看过往对话
- **智能召回**: 自动匹配相关历史上下文
- **向量数据库**: Qdrant 集成，语义检索
- **用户档案**: 记住编码习惯和项目历史

**预计命令**:
```bash
/history [n]     # 查看最近 n 条对话
/search <query>  # 搜索历史对话
/recall          # 智能召回相关记忆
↑/↓             # 滚动查看历史（类似 shell history）
```

### 4. 自主学习与技能进化 (v0.3.0 - v0.6.0)
- Agent 自动学习新技能
- 从成功案例提取模式，生成新 Skill
- 强化学习：根据用户反馈（接受/拒绝）优化
- 元学习：从历史任务提取通用模式

### 5. 企业级功能 (v0.4.0 - v0.7.0)
- **Skill 市场**: 共享和发布团队专用 Skill
- **权限审计**: 敏感操作需要审批，全量追溯
- **团队协作**: 多用户共享配置和记忆

**预计命令**:
```bash
jiasinecli skill search <keyword>
jiasinecli skill install <name>
jiasinecli skill publish <path>
```

### 6. 多模态能力 (v0.5.0 - v0.8.0)
- **视觉理解**: 截图分析、架构图解析、代码识别
- **语音交互**: 语音命令 (Whisper ASR) + 语音输出 (TTS)
- **视频理解**: 从技术视频提取代码示例
- **文档解析**: PDF/Word/PPT 智能解析

**预计命令**:
```bash
jiasinecli vision analyze screenshot.png
jiasinecli vision code-from-image code_screenshot.jpg
jiasinecli voice ask "创建一个 REST API 服务"
jiasinecli video extract-code tutorial.mp4
```

### 7. 智能化开发工作流 (v0.6.0 - v0.9.0)
- 从技术视频提取代码示例
- 解析 PDF 文档生成 API 客户端
- 分析架构图生成项目脚手架

**预计工作流**:
```bash
jiasinecli workflow run video-to-code --input tutorial.mp4
jiasinecli workflow run pdf-to-client --input api_spec.pdf
jiasinecli workflow run diagram-to-scaffold --input architecture.png
```

### 8. 代码健康度监控 (v0.5.0 - v0.8.0)
- 后台持续分析代码库
- 主动提醒技术债务和问题
- 智能 Code Review
- Git hook 集成
- 自动化测试生成

**预计命令**:
```bash
jiasinecli monitor start --interval 1h
jiasinecli monitor report
jiasinecli monitor install-hooks
jiasinecli test generate --file service.go
```

### 9. 沙箱安全系统 (v0.6.0 - v0.9.0)
- 隔离环境运行程序
- 文件系统虚拟化
- 网络隔离
- 安全测试和模拟

**预计命令**:
```bash
jiasinecli sandbox create --name test-env
jiasinecli sandbox run --name test-env -- go test ./...
jiasinecli sandbox destroy --name test-env
```

---

## 时间线

| 版本 | 预计发布 | 主要功能 |
|------|----------|----------|
| v0.1.1 | 2026-03 | ✅ PowerShell 背景色优化 |
| v0.2.0 | 2026-04 | 基础 Agent 框架、SQLite 历史 |
| v0.3.0 | 2026-05 | 任务分解、反馈系统 |
| v0.4.0 | 2026-07 | 向量数据库、Skill 市场 |
| v0.5.0 | 2026-09 | 视觉理解、代码监控 |
| v0.6.0 | 2026-11 | 语音交互、沙箱系统 |
| v0.7.0 | 2027-01 | 团队协作、视频理解 |
| v0.8.0 | 2027-04 | 自动测试、文档解析 |
| v0.9.0 | 2027-07 | 完整企业级功能 |
| v1.0.0 | 2027-10 | 正式版本发布 🎉 |

---

## 如何参与

### 功能投票
在 GitHub Issues 中为您最期待的功能投票 (👍)：
- [#2 多 Agent 协作系统](https://github.com/xiangjianhe-github/jiasinecli/issues/2)
- [#3 历史对话查看](https://github.com/xiangjianhe-github/jiasinecli/issues/3)
- [#4 视觉理解能力](https://github.com/xiangjianhe-github/jiasinecli/issues/4)

### 贡献代码
欢迎提交 PR！请参考：
- [开发指南](./CONTRIBUTING.md)
- [代码规范](./CODE_STYLE.md)

### 反馈建议
- 提交 Issue: [GitHub Issues](https://github.com/xiangjianhe-github/jiasinecli/issues)
- 讨论区: [GitHub Discussions](https://github.com/xiangjianhe-github/jiasinecli/discussions)

---

## 当前状态

- 🏗️ **开发中**: 多 Agent 框架 (v0.2.0)
- 📝 **规划中**: 历史对话查看 UI
- 🔬 **研究中**: 向量数据库技术选型

---

## 技术细节

完整的技术规划和架构设计请参考：
- [长期路线图](./LONG_TERM_ROADMAP.md) - 详细功能规划
- [架构设计](./ARCHITECTURE.md) - 系统架构文档
- [API 文档](./API.md) - 接口说明

---

**下一步**: 开始实现 v0.2.0 的基础 Agent 框架！🚀
