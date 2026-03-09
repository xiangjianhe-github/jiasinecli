# 更新地址变更说明

## 🔄 变更内容

### 下载地址变更
- **旧地址**（GitLab）: `https://gitlab.gz.cvte.cn/mnt/jiasine/jiasinecli/-/raw/main/dist/`
- **新地址**（网盘）: `https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/`

### 版本配置
- **版本信息**: `https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/version.json`

## 📦 二进制文件命名规则

| 平台 | 文件名 |
|------|--------|
| Windows amd64 | `jiasinecli-windows.exe` |
| Windows arm64 | `jiasinecli-windows-arm64.exe` |
| Linux amd64 | `jiasinecli-linux` |
| Linux arm64 | `jiasinecli-linux-arm64` |
| Linux arm (树莓派) | `jiasinecli-raspi` |
| macOS Apple Silicon | `jiasinecli-macos-arm` |
| macOS Intel | `jiasinecli-macos-intel` |

## 🔧 实现细节

### 平台自动检测
应用会自动检测当前运行平台（OS + 架构），并下载对应的二进制文件：

```go
// 获取平台标识：windows-amd64, linux-amd64, darwin-arm64 等
platform := runtime.GOOS + "-" + runtime.GOARCH

// 根据平台映射到对应的文件名
fileNameMap := map[string]string{
    "windows-amd64": "jiasinecli-windows.exe",
    "linux-amd64":   "jiasinecli-linux",
    "darwin-arm64":  "jiasinecli-macos-arm",
    // ...
}
```

### 下载逻辑
1. 检查 `version.json` 中的 `download_urls` 字段
2. 如果存在对应平台的 URL，使用指定 URL
3. 如果不存在，使用默认规则拼接：`基础URL + 文件名`

### 示例 URL
```
# Windows 用户下载
https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-windows.exe

# Linux 用户下载
https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-linux

# macOS Apple Silicon 用户下载
https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-macos-arm
```

## ✅ 测试结果

所有平台映射已通过单元测试：
```
✓ windows-amd64 → jiasinecli-windows.exe
✓ windows-arm64 → jiasinecli-windows-arm64.exe
✓ linux-amd64 → jiasinecli-linux
✓ linux-arm64 → jiasinecli-linux-arm64
✓ linux-arm → jiasinecli-raspi
✓ darwin-arm64 → jiasinecli-macos-arm
✓ darwin-amd64 → jiasinecli-macos-intel
```

## 📝 更新后的配置示例

`version.json` 配置：
```json
{
  "major": 0,
  "minor": 1,
  "patch": 0,
  "download_urls": {
    "windows-amd64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-windows.exe",
    "windows-arm64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-windows-arm64.exe",
    "linux-amd64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-linux",
    "linux-arm64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-linux-arm64",
    "linux-arm": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-raspi",
    "darwin-amd64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-macos-intel",
    "darwin-arm64": "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-macos-arm"
  },
  "release_date": "2026-03-06",
  "changelog": "更新内容..."
}
```

## 🚀 部署步骤

1. 编译所有平台的二进制文件：
   ```bash
   python build.py cross
   ```

2. 将 `dist/` 目录中的文件按新命名规则上传到网盘：
   - `dist/jiasinecli-windows.exe` → 上传为 `jiasinecli-windows.exe`
   - `dist/jiasinecli-windows-arm64.exe` → 上传为 `jiasinecli-windows-arm64.exe`
   - `dist/jiasinecli-linux` → 上传为 `jiasinecli-linux`
   - `dist/jiasinecli-linux-arm64` → 上传为 `jiasinecli-linux-arm64`
   - `dist/jiasinecli-raspi` → 上传为 `jiasinecli-raspi`
   - `dist/jiasinecli-macos-arm` → 上传为 `jiasinecli-macos-arm`
   - `dist/jiasinecli-macos-intel` → 上传为 `jiasinecli-macos-intel`

3. 更新 `version.json` 并上传到网盘根目录

4. 用户执行 `jiasinecli update` 即可自动更新
