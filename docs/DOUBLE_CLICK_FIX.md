# 双击启动修复测试说明

## 🔧 修复内容

### 1. **核心问题修复**
- ✅ 修复双击 `jiasinecli-windows.exe` 时的闪退问题
- ✅ 改用 **PowerShell** 管理员模式（替代 Windows Terminal）
- ✅ 用户取消 UAC 提示时，程序在当前环境继续运行（不闪退）
- ✅ 已是管理员权限时，直接运行（不重启）

### 2. **跨平台兼容性**
- ✅ 所有 7 个平台编译通过
  - Windows (amd64, arm64)
  - Linux (amd64, arm64, arm)
  - macOS (amd64, arm64)

## 🧪 测试步骤

### 测试场景 1：双击启动（普通用户）

1. **在文件资源管理器中双击** `jiasinecli.exe`
2. **预期行为**：
   - 弹出 UAC 提示："是否允许此应用对设备进行更改？"
   - 显示："正在请求管理员权限..."

3. **选择"是"**：
   - 新的 PowerShell 窗口以管理员权限打开
   - 进入交互式 Shell 模式
   - 原窗口关闭

4. **选择"否"（取消）**：
   - ❌ **旧版本**：闪退（窗口直接关闭）
   - ✅ **新版本**：显示警告信息，在当前窗口继续运行

   ```
   ⚠ 未获得管理员权限，将在当前环境继续运行
   某些功能可能受限，建议右键选择【以管理员身份运行】

   Jiasine CLI Shell - 交互式模式
   输入 'help' 查看帮助，'exit' 退出

   jiasine>
   ```

### 测试场景 2：已是管理员权限

1. **右键 → 以管理员身份运行** `jiasinecli.exe`
2. **预期行为**：
   - 不弹出 UAC 提示
   - 不显示"正在请求管理员权限..."
   - 直接进入交互式 Shell

### 测试场景 3：从 PowerShell 启动（普通用户）

1. **打开 PowerShell**
2. **运行**：`.\jiasinecli.exe`
3. **预期行为**：
   - 不弹出 UAC 提示（不强制要求管理员权限）
   - 直接进入交互式 Shell

### 测试场景 4：从 PowerShell 启动（管理员）

1. **以管理员身份打开 PowerShell**
2. **运行**：`.\jiasinecli.exe`
3. **预期行为**：
   - 直接进入交互式 Shell

### 测试场景 5：命令行模式

1. **PowerShell 中运行**：`.\jiasinecli.exe version`
2. **预期行为**：
   - 显示版本信息
   - 不弹出 UAC 提示
   - 不请求管理员权限

## 📝 代码变更摘要

### 1. `internal/shell/shell_windows.go`

**旧逻辑**：
```go
func RelaunchInWindowsTerminalAdmin() {
    // 尝试启动，无论成功失败都 return
    // 调用者直接 return，程序退出 → 闪退
}
```

**新逻辑**：
```go
func RelaunchInWindowsTerminalAdmin() bool {
    // 已是管理员 → 返回 false，继续运行
    if isAdmin() {
        return false
    }

    // 不是双击启动 → 返回 false，继续运行
    if !IsDoubleClicked() {
        return false
    }

    // 尝试启动 PowerShell 管理员
    success := launchInPowerShellAdmin(exe)
    if !success {
        // 用户取消 → 显示警告，返回 false，继续运行
        fmt.Println("⚠ 未获得管理员权限，将在当前环境继续运行")
        return false
    }

    // 成功启动 → 返回 true，退出当前进程
    return true
}
```

### 2. `main.go`

**旧逻辑**：
```go
if len(os.Args) <= 1 && shell.IsDoubleClicked() {
    shell.RelaunchInWindowsTerminalAdmin()
    return  // 无条件退出 → 闪退
}
```

**新逻辑**：
```go
if len(os.Args) <= 1 && shell.IsDoubleClicked() {
    if shell.RelaunchInWindowsTerminalAdmin() {
        // 仅在成功重新启动时退出
        return
    }
    // 否则继续运行
}
```

### 3. `internal/shell/shell_other.go`

```go
// 跨平台兼容：非 Windows 平台返回 false
func RelaunchInWindowsTerminalAdmin() bool {
    return false
}
```

## ✅ 验证清单

- [x] 本地编译成功 (Windows amd64)
- [x] 跨平台编译成功 (7/7 平台)
- [ ] 测试双击启动 + 确认 UAC
- [ ] 测试双击启动 + 取消 UAC（不闪退）
- [ ] 测试右键管理员运行
- [ ] 测试 PowerShell 启动
- [ ] 测试命令行参数（不弹 UAC）

## 🎯 预期改进

| 场景 | 旧版本行为 | 新版本行为 |
|------|-----------|-----------|
| 双击 + 取消 UAC | ❌ 闪退 | ✅ 显示警告，继续运行 |
| 双击 + 确认 UAC | ✅ 在新窗口打开 | ✅ 在 PowerShell 管理员打开 |
| 已是管理员 | ⚠ 仍尝试重启 | ✅ 直接运行 |
| 终端启动 | ⚠ 可能提示 UAC | ✅ 不提示，直接运行 |
| 带参数启动 | ✅ 正常 | ✅ 正常 |

## 🚀 部署建议

1. **重新编译所有平台**：
   ```bash
   python build.py cross
   ```

2. **上传到网盘**：
   - 将 `dist/` 目录中的所有文件上传到：
   - `https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/`

3. **更新 version.json**：
   - 增加补丁版本号（如 0.1.1）
   - 更新 changelog 说明修复内容

## 📖 用户体验提升

### 用户取消 UAC 时的友好提示：
```
正在请求管理员权限...

⚠ 未获得管理员权限，将在当前环境继续运行
某些功能可能受限，建议右键选择【以管理员身份运行】

Jiasine CLI Shell - 交互式模式
输入 'help' 查看帮助，'exit' 退出

jiasine>
```

**关键改进**：
- ✅ 不再闪退
- ✅ 明确告知状态
- ✅ 提供操作建议
- ✅ 允许在受限模式继续使用
