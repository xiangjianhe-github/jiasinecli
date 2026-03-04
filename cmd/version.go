package cmd

import (
	"fmt"
	"runtime"

	"github.com/xiangjianhe-github/jiasinecli/internal/version"
	"github.com/spf13/cobra"
)

// 构建时通过 -ldflags 注入
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

var versionShort bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long: `显示 Jiasine CLI 版本信息。

版本管控规则 (SemVer 2.0):
  MAJOR  — 不兼容的 API 变更 (命令移除/重命名, 配置格式变更)
  MINOR  — 向后兼容的功能新增 (新命令, 新配置选项)
  PATCH  — 向后兼容的问题修复 (bug fix, 性能优化)

预发布标签:
  alpha.N  — 内部测试
  beta.N   — 公开测试
  rc.N     — 候选发布`,
	Run: func(cmd *cobra.Command, args []string) {
		// 用构建注入的值更新版本信息
		ver := version.Current
		ver.GitCommit = GitCommit
		ver.BuildDate = BuildDate
		ver.GoVersion = runtime.Version()
		ver.Platform = runtime.GOOS + "/" + runtime.GOARCH

		// 如果构建时注入了版本号，解析覆盖
		if Version != "dev" && Version != "" {
			if parsed, err := version.Parse(Version); err == nil {
				parsed.GitCommit = GitCommit
				parsed.BuildDate = BuildDate
				parsed.GoVersion = runtime.Version()
				parsed.Platform = runtime.GOOS + "/" + runtime.GOARCH
				ver = parsed
			}
		}

		if versionShort {
			fmt.Println(ver.String())
			return
		}

		fmt.Printf("Jiasine CLI %s\n", ver.String())
		fmt.Printf("  语义版本: %d.%d.%d\n", ver.Major, ver.Minor, ver.Patch)
		if ver.PreRelease != "" {
			fmt.Printf("  预发布:   %s\n", ver.PreRelease)
		}
		fmt.Printf("  Git:      %s\n", ver.GitCommit)
		fmt.Printf("  构建日期: %s\n", ver.BuildDate)
		fmt.Printf("  Go:       %s\n", ver.GoVersion)
		fmt.Printf("  平台:     %s\n", ver.Platform)
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "仅输出版本号")
	rootCmd.AddCommand(versionCmd)
}
