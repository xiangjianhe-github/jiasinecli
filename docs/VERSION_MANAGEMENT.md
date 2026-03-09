# 版本管理与自动更新

## 版本配置位置

### 1. 代码中的版本定义
**文件**: [`internal/version/version.go`](internal/version/version.go#L56-L61)

```go
// 当前版本 — 发布时手动更新此处
var Current = Info{
	Major:      0,
	Minor:      1,
	Patch:      0,
	PreRelease: "alpha.1",
}
```

### 2. 服务器版本配置
**文件**: `version.json` （需上传到 https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA）

示例配置：
```json
{
  "major": 0,
  "minor": 1,
  "patch": 0,
  "pre_release": "alpha.1",
  "build_meta": "",
  "git_commit": "abc1234",
  "build_date": "2026-03-06T10:00:00Z",
  "go_version": "go1.26.0",
  "platform": "multi",
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
  "changelog": "## v0.1.0-alpha.1\n\n### 新增功能\n- AI 交互模式优化"
}
```

## 版本规范（SemVer 2.0）

### 版本格式
```
vMAJOR.MINOR.PATCH[-prerelease][+buildmetadata]
```

### 版本号说明
- **MAJOR（主版本）**: 不兼容的 API 变更
  - 移除或重命名命令/子命令
  - 修改配置文件格式（不向后兼容）
  - 修改插件接口/桥接层协议

- **MINOR（次版本）**: 向后兼容的功能新增
  - 新增命令/子命令
  - 新增配置选项（带默认值）
  - 新增桥接/服务类型支持

- **PATCH（补丁版本）**: 向后兼容的问题修复
  - 修复 bug
  - 性能优化
  - 文档更新

### 预发布标识（prerelease）
- `alpha.N` — 内部测试版
- `beta.N` — 公测版
- `rc.N` — 候选发布版

### 构建元数据（buildmetadata）
- Git commit hash
- 构建日期
- CI 构建编号

### 版本示例
```
v0.1.0-alpha.1      # 内测版本
v0.1.0-beta.2       # 公测版本
v0.1.0-rc.1         # 候选发布
v0.1.0              # 正式版本
v0.1.0+20060102     # 带构建日期
v1.0.0-beta.1+abc123  # 完整格式
```

## 发布新版本流程

### 1. 更新代码版本号
编辑 `internal/version/version.go`:
```go
var Current = Info{
	Major:      0,
	Minor:      2,  // 次版本号 +1
	Patch:      0,
	PreRelease: "", // 移除预发布标识（正式版）
}
```

### 2. 构建跨平台二进制文件
```bash
python build.py cross
```

生成的文件位于 `dist/` 目录：
- `jiasinecli-windows.exe` (Windows amd64)
- `jiasinecli-windows-arm64.exe` (Windows ARM64)
- `jiasinecli-linux` (Linux amd64)
- `jiasinecli-linux-arm64` (Linux ARM64)
- `jiasinecli-macos-arm` (macOS Apple Silicon)
- `jiasinecli-macos-intel` (macOS Intel)

### 3. 上传二进制文件到服务器
将编译好的二进制文件上传到：
```
https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/
```

文件命名规则：
- Windows amd64: `jiasinecli-windows.exe`
- Windows arm64: `jiasinecli-windows-arm64.exe`
- Linux amd64: `jiasinecli-linux`
- Linux arm64: `jiasinecli-linux-arm64`
- Linux arm (Raspberry Pi): `jiasinecli-raspi`
- macOS Apple Silicon: `jiasinecli-macos-arm`
- macOS Intel: `jiasinecli-macos-intel`

### 4. 更新 version.json
编辑 `version.json`，更新版本号和下载链接：
```json
{
  "major": 0,
  "minor": 2,
  "patch": 0,
  "pre_release": "",
  "git_commit": "最新提交的 commit hash",
  "build_date": "2026-03-06T10:00:00Z",
  "download_urls": {
    "windows-amd64": "https://gitlab.gz.cvte.cn/mnt/jiasine/jiasinecli/-/raw/main/dist/windows-amd64/jiasinecli.exe",
    ...
  },
  "release_date": "2026-03-06",
  "changelog": "## v0.2.0\n\n### 新增功能\n- 功能描述"
}
```

### 5. 上传 version.json 到服务器
将 `version.json` 上传到:
```
https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/version.json
```

## 自动更新机制

### 启动时自动检查
应用启动时会在后台异步检查更新，如果有新版本会提示：
```
💡 发现新版本 v0.2.0，运行 'jiasinecli update' 更新
```

### 手动检查更新
```bash
# 检查并自动更新
jiasinecli update

# 仅检查，不执行更新
jiasinecli update --check

# 强制重新下载当前版本
jiasinecli update --force
```

### 更新流程
1. 从服务器获取 `version.json`
2. 比较版本号（遵循 SemVer 规则）
3. 如果有新版本：
   - 显示版本信息和更新日志
   - 根据当前平台下载对应的二进制文件
   - 备份当前版本（.old 后缀）
   - 替换为新版本
   - 提示重启应用

## 配置方法总结

1. **本地开发版本**: 修改 `internal/version/version.go` 中的 `Current` 变量
2. **发布版本**: 在 `version.json` 中配置，并上传到服务器
3. **构建时注入**: 通过 `-ldflags` 参数注入 Git commit 和构建日期（已在 `build.py` 中配置）

## 注意事项

1. 版本号必须严格遵循 SemVer 2.0 规范
2. 主版本号为 0 表示初始开发阶段，API 不保证稳定
3. 预发布版本（alpha、beta、rc）会被认为低于正式版本
4. 更新前会自动备份旧版本（.old 后缀）
5. 下载失败时不会影响当前版本的使用
