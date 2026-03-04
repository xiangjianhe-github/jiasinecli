// Package version 提供语义化版本管控
// 遵循 SemVer 2.0 规范: MAJOR.MINOR.PATCH[-prerelease]
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// 版本管控规则:
//
// 格式: vMAJOR.MINOR.PATCH[-prerelease][+build]
//
// MAJOR (主版本): 不兼容的 API 变更
//   - 移除或重命名命令/子命令
//   - 修改配置文件格式 (不向后兼容)
//   - 修改插件接口/桥接层协议
//
// MINOR (次版本): 向后兼容的功能新增
//   - 新增命令/子命令
//   - 新增配置选项 (带默认值)
//   - 新增桥接/服务类型支持
//
// PATCH (补丁版本): 向后兼容的问题修复
//   - 修复 bug
//   - 性能优化
//   - 文档更新
//
// PreRelease 标签:
//   - alpha.N  — 内部测试版
//   - beta.N   — 公测版
//   - rc.N     — 候选发布版
//
// 示例:
//   v1.0.0             — 正式版
//   v1.1.0-alpha.1     — 1.1 新功能内测
//   v1.1.0-beta.2      — 1.1 公测第2版
//   v1.1.0-rc.1        — 1.1 候选发布
//   v2.0.0             — 重大变更

// Info 语义化版本信息
type Info struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"pre_release,omitempty"` // alpha.1, beta.2, rc.1
	BuildMeta  string `json:"build_meta,omitempty"`  // 构建元数据
	GitCommit  string `json:"git_commit,omitempty"`
	BuildDate  string `json:"build_date,omitempty"`
	GoVersion  string `json:"go_version,omitempty"`
	Platform   string `json:"platform,omitempty"` // GOOS/GOARCH
}

// 当前版本 — 发布时手动更新此处
var Current = Info{
	Major:      0,
	Minor:      1,
	Patch:      0,
	PreRelease: "alpha.1",
}

// String 返回完整版本字符串
func (v Info) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	if v.BuildMeta != "" {
		s += "+" + v.BuildMeta
	}
	return s
}

// Short 返回短版本 (不含预发布和构建元数据)
func (v Info) Short() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsPreRelease 是否为预发布版本
func (v Info) IsPreRelease() bool {
	return v.PreRelease != ""
}

// IsCompatible 检查给定版本是否与当前版本 API 兼容 (同一主版本)
func (v Info) IsCompatible(other Info) bool {
	return v.Major == other.Major
}

// Compare 比较两个版本: -1 (v < other), 0 (v == other), 1 (v > other)
func (v Info) Compare(other Info) int {
	if v.Major != other.Major {
		return intCompare(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return intCompare(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return intCompare(v.Patch, other.Patch)
	}

	// 无预发布 > 有预发布 (正式版高于预发布)
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1
	}

	return strings.Compare(v.PreRelease, other.PreRelease)
}

// Parse 从字符串解析版本
func Parse(s string) (Info, error) {
	s = strings.TrimPrefix(s, "v")

	var info Info

	// 分离构建元数据
	if idx := strings.Index(s, "+"); idx >= 0 {
		info.BuildMeta = s[idx+1:]
		s = s[:idx]
	}

	// 分离预发布标签
	if idx := strings.Index(s, "-"); idx >= 0 {
		info.PreRelease = s[idx+1:]
		s = s[:idx]
	}

	// 解析 MAJOR.MINOR.PATCH
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Info{}, fmt.Errorf("无效的版本格式: 需要 MAJOR.MINOR.PATCH, 得到 %q", s)
	}

	var err error
	info.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return Info{}, fmt.Errorf("无效的主版本号: %w", err)
	}
	info.Minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return Info{}, fmt.Errorf("无效的次版本号: %w", err)
	}
	info.Patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return Info{}, fmt.Errorf("无效的补丁版本号: %w", err)
	}

	return info, nil
}

func intCompare(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}
