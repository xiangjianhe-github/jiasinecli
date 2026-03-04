package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xiangjianhe-github/jiasinecli/internal/testrunner"
	"github.com/spf13/cobra"
)

var testLang string

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "运行多语言集成测试",
	Long: `运行 Jiasine CLI 与各语言后端的集成测试，验证桥接层和服务层功能。

支持测试:
  c           — C 动态库 FFI 测试 (DLL/SO)
  python      — Python HTTP 服务 + 进程调用测试
  rust        — Rust 动态库 FFI 测试
  csharp      — C# HTTP 服务测试
  js          — JavaScript HTTP 服务 + 进程调用测试 (Node.js)
  typescript  — TypeScript HTTP 服务 + 进程调用测试 (tsx)
  java        — Java HTTP 服务 + 进程调用测试 (JDK)
  swift       — Swift 编译 + 进程调用测试
  objc        — Objective-C 动态库 FFI 测试
  all         — 运行所有可用测试`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 确定测试目录
		execPath, _ := os.Executable()
		baseDir := filepath.Dir(filepath.Dir(execPath))
		testsDir := filepath.Join(baseDir, "tests")

		// 如果从项目目录运行，优先用 cwd
		if cwd, err := os.Getwd(); err == nil {
			localTests := filepath.Join(cwd, "tests")
			if _, err := os.Stat(localTests); err == nil {
				testsDir = localTests
			}
		}

		runner := testrunner.New(testsDir)

		if testLang == "" || testLang == "all" {
			return runner.RunAll()
		}

		return runner.RunLanguage(testLang)
	},
}

var testStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查各语言测试环境就绪状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 确定测试目录
		testsDir := "tests"
		if cwd, err := os.Getwd(); err == nil {
			testsDir = filepath.Join(cwd, "tests")
		}

		runner := testrunner.New(testsDir)
		status := runner.CheckEnvironment()

		fmt.Println("═══════════════════════════════════════════")
		fmt.Println("  Jiasine CLI 多语言测试环境检查")
		fmt.Println("═══════════════════════════════════════════")

		for _, s := range status {
			icon := "✗"
			if s.Ready {
				icon = "✓"
			}
			fmt.Printf("  %s %-10s %-10s %s\n", icon, s.Language, s.Status, s.Detail)
		}
		fmt.Println("═══════════════════════════════════════════")

		return nil
	},
}

func init() {
	testCmd.Flags().StringVarP(&testLang, "lang", "l", "all", "指定测试语言 (c/python/rust/csharp/js/typescript/java/swift/objc/all)")
	testCmd.AddCommand(testStatusCmd)
	rootCmd.AddCommand(testCmd)
}
