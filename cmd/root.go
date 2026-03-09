package cmd

import (
	"fmt"
	"os"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"github.com/xiangjianhe-github/jiasinecli/internal/theme"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	verbose     bool
	Interactive bool
)

// rootCmd 是 CLI 的根命令
var rootCmd = &cobra.Command{
	Use:   "jiasinecli",
	Short: "Jiasine CLI - Cross-platform multi-language support system",
	Long: banner.Logo() + `

架构设计：
  • CLI 层 (Go)      - 命令解析、并发控制、用户体验
  • 桥接层 (Bridge)   - FFI 调用动态库 (C/Rust/Obj-C/.NET AOT)
  • 服务层 (Service)  - HTTP/进程调用独立服务 (Python/C#/JS/TS/Java/Swift)
  • 插件层 (Plugin)   - 可扩展的插件系统

支持语言: C · Python · Rust · C# · JavaScript · TypeScript · Java · Swift · Objective-C
支持平台: Windows / macOS / Linux`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeApp()
	},
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteArgs 用指定参数执行命令（交互式 Shell 使用）
func ExecuteArgs(args []string) error {
	// 重置各命令的标志到默认值，防止前次执行的标志残留
	resetFlags()

	// 静默 Cobra 自身的错误和用法输出，交互模式由 shell 自行处理
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	// 重置 args 以便下次调用
	rootCmd.SetArgs(nil)
	return err
}

// resetFlags 重置所有命令标志到默认值
// Cobra 在重复执行时不会自动重置未出现的标志
func resetFlags() {
	versionShort = false
	testLang = "all"
	aiProvider = ""
	aiModel = ""
	aiAgent = ""
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认 $HOME/.jiasine/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出模式")
	rootCmd.PersistentFlags().BoolVarP(&Interactive, "interactive", "i", false, "进入交互式 Shell 模式")

	// 绑定 viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

var appInitialized bool

func initializeApp() error {
	if appInitialized {
		return nil
	}

	// 初始化配置
	if err := config.Init(cfgFile); err != nil {
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	// 从配置加载主题
	theme.Set(theme.ThemeName(config.GetTheme()))
	banner.RefreshColors()

	// 初始化日志
	if err := logger.Init(verbose); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}

	// 确保必要目录存在
	homeDir, _ := os.UserHomeDir()
	dirs := []string{
		homeDir + "/.jiasine",
		homeDir + "/.jiasine/plugins",
		homeDir + "/.jiasine/logs",
		homeDir + "/.jiasine/agents",
		homeDir + "/.jiasine/skills",
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	appInitialized = true
	return nil
}
