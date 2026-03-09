package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/shell"
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
		fmt.Printf("\n%s🔧 Jiasine CLI 环境配置%s\n\n", banner.Bold+banner.BrightCyan, banner.Reset)

		if runtime.GOOS == "windows" {
			return setupWindows()
		}
		return setupOther()
	},
}

func setupWindows() error {
	// 使用 InstallToPath 自动复制并添加到 PATH
	installDir, err := shell.InstallToPath()
	if err != nil {
		return fmt.Errorf("安装失败: %w", err)
	}

	fmt.Printf("  %s✓%s 程序已安装到: %s\n", banner.BrightGreen, banner.Reset, installDir)
	fmt.Printf("  %s✓%s 已添加到用户 PATH（新窗口生效）\n", banner.BrightGreen, banner.Reset)

	// 创建 jiasine.cmd 别名
	aliasPath := filepath.Join(installDir, "jiasine.cmd")
	exePath := filepath.Join(installDir, "jiasinecli.exe")
	aliasContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", exePath)

	if err := os.WriteFile(aliasPath, []byte(aliasContent), 0755); err != nil {
		fmt.Printf("  %s⚠%s 创建 jiasine.cmd 别名失败: %v\n", banner.Yellow, banner.Reset, err)
	} else {
		fmt.Printf("  %s✓%s 已创建 jiasine.cmd 别名\n", banner.BrightGreen, banner.Reset)
	}

	fmt.Printf("\n%s配置完成！%s 请打开新的 PowerShell 窗口，然后输入:\n", banner.Bold, banner.Reset)
	fmt.Printf("    %sjiasinecli%s       # 启动 AI 模式\n", banner.BrightGreen, banner.Reset)
	fmt.Printf("    %sjiasine%s          # 同上（别名）\n", banner.BrightGreen, banner.Reset)
	fmt.Println()
	return nil
}

func setupOther() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	exeDir := filepath.Dir(exePath)

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
		fmt.Printf("\n  请手动添加：\n")
		fmt.Printf("  %sexport PATH=\"$PATH:%s\"%s\n", banner.Dim, exeDir, banner.Reset)
		fmt.Printf("  %s# 添加到 ~/.bashrc 或 ~/.zshrc 使其永久生效%s\n", banner.Dim, banner.Reset)
	}

	fmt.Println()
	return nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
