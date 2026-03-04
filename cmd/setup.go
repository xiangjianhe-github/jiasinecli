package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "配置系统环境 (PATH、别名)",
	Long: `将 jiasinecli 添加到系统 PATH 并创建 jiasine 别名。

执行后，可在任意 PowerShell / CMD 窗口中直接使用:
  jiasine          # 启动交互模式
  jiasinecli       # 同上
  jiasine ai chat  # 直接使用子命令`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("获取程序路径失败: %w", err)
		}

		exeDir := filepath.Dir(exePath)
		exeName := filepath.Base(exePath) // jiasinecli.exe

		fmt.Printf("\n%s🔧 Jiasine CLI 环境配置%s\n\n", banner.Bold+banner.BrightCyan, banner.Reset)

		// 1. 检查是否已在 PATH 中
		pathEnv := os.Getenv("PATH")
		inPath := false
		for _, p := range filepath.SplitList(pathEnv) {
			if strings.EqualFold(p, exeDir) {
				inPath = true
				break
			}
		}

		if inPath {
			fmt.Printf("  %s✓%s 已在 PATH 中: %s\n", banner.BrightGreen, banner.Reset, exeDir)
		} else {
			fmt.Printf("  %s⚠%s 未在 PATH 中: %s\n", banner.Yellow, banner.Reset, exeDir)
			fmt.Printf("\n  请手动添加到 PATH (管理员权限):\n")
			fmt.Printf("  %s[Environment]::SetEnvironmentVariable('PATH', $env:PATH + ';%s', 'User')%s\n\n",
				banner.Dim, exeDir, banner.Reset)
		}

		// 2. 创建 jiasine.cmd 别名（同目录下）
		aliasPath := filepath.Join(exeDir, "jiasine.cmd")
		aliasContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", filepath.Join(exeDir, exeName))

		if _, err := os.Stat(aliasPath); err == nil {
			fmt.Printf("  %s✓%s jiasine.cmd 别名已存在\n", banner.BrightGreen, banner.Reset)
		} else {
			if err := os.WriteFile(aliasPath, []byte(aliasContent), 0755); err != nil {
				fmt.Printf("  %s✗%s 创建 jiasine.cmd 失败: %v\n", banner.Yellow, banner.Reset, err)
			} else {
				fmt.Printf("  %s✓%s 已创建 jiasine.cmd 别名\n", banner.BrightGreen, banner.Reset)
			}
		}

		// 3. 创建 PowerShell profile 别名
		fmt.Printf("\n%s配置说明:%s\n", banner.Bold, banner.Reset)
		fmt.Println()

		if inPath {
			fmt.Printf("  现在可以在任意终端使用:\n")
			fmt.Printf("    %sjiasine%s          # 启动交互模式\n", banner.BrightGreen, banner.Reset)
			fmt.Printf("    %sjiasinecli%s       # 同上\n", banner.BrightGreen, banner.Reset)
			fmt.Printf("    %sjiasine ai chat%s  # 直接使用命令\n", banner.BrightGreen, banner.Reset)
		} else {
			fmt.Printf("  添加 PATH 后，可在任意终端使用:\n")
			fmt.Printf("    %sjiasine%s          # 启动交互模式\n", banner.BrightGreen, banner.Reset)
			fmt.Printf("    %sjiasinecli%s       # 同上\n", banner.BrightGreen, banner.Reset)
		}

		fmt.Printf("\n  或添加 PowerShell 别名 (一次性设置):\n")
		fmt.Printf("  %sAdd-Content $PROFILE 'Set-Alias jiasine \"%s\"'%s\n",
			banner.Dim, filepath.Join(exeDir, exeName), banner.Reset)

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
