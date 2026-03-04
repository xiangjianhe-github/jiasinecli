package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"github.com/xiangjianhe-github/jiasinecli/internal/plugin"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "插件管理",
	Long: `管理 Jiasine CLI 插件

插件存放在应用目录下的 plugin/ 子目录中。
每个插件是一个目录，包含 <PluginName>.json 描述文件。

示例:
  jiasinecli plugin view                 # 查看插件市场 (列出所有可用插件)
  jiasinecli plugin open SerialTool      # 打开 SerialTool 插件
  jiasinecli plugin list                 # 列出已安装的插件
  jiasinecli plugin install MyPlugin     # 创建新插件骨架
  jiasinecli plugin remove MyPlugin      # 卸载插件`,
}

// plugin view  —— 插件市场：扫描 plugin/ 目录，展示所有可用插件
var pluginViewCmd = &cobra.Command{
	Use:   "view",
	Short: "查看插件市场 (列出所有可用插件)",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		plugins, err := mgr.Scan()
		if err != nil {
			return fmt.Errorf("扫描插件目录失败: %w", err)
		}

		fmt.Printf("\n%s%s 插件市场%s  %s(%s)%s\n",
			banner.Bold+banner.BrightCyan, "📦", banner.Reset,
			banner.Dim, mgr.PluginDir(), banner.Reset)
		fmt.Println(strings.Repeat("─", 70))

		if len(plugins) == 0 {
			fmt.Printf("\n  %s暂无可用插件%s\n", banner.Dim, banner.Reset)
			fmt.Printf("  将插件目录放入 %s%s%s 即可被发现\n",
				banner.BrightGreen, mgr.PluginDir(), banner.Reset)
			fmt.Printf("  格式: plugin/<名称>/<名称>.json\n\n")
			return nil
		}

		for i, p := range plugins {
			icon := p.Icon
			if icon == "" {
				icon = "🔌"
			}

			// 状态
			status := fmt.Sprintf("%s启用%s", banner.BrightGreen, banner.Reset)
			if !p.Enabled {
				status = fmt.Sprintf("%s禁用%s", banner.Dim, banner.Reset)
			}

			// 入口
			entry := p.EntryPoint
			if entry == "" {
				entry = "-"
			}

			fmt.Printf("\n  %s %s%s%s  %sv%s%s  [%s]\n",
				icon,
				banner.Bold+banner.BrightCyan, p.Name, banner.Reset,
				banner.Dim, p.Version, banner.Reset,
				status)

			if p.Description != "" {
				fmt.Printf("    %s%s%s\n", banner.Dim, p.Description, banner.Reset)
			}
			if p.Author != "" {
				fmt.Printf("    作者: %s\n", p.Author)
			}
			if entry != "-" {
				fmt.Printf("    入口: %s%s%s\n", banner.BrightGreen, entry, banner.Reset)
			}
			if len(p.Tags) > 0 {
				fmt.Printf("    标签: %s\n", strings.Join(p.Tags, ", "))
			}

			if i < len(plugins)-1 {
				fmt.Println()
			}
		}

		fmt.Printf("\n%s提示%s: 使用 %splugin <名称>%s 打开插件\n\n",
			banner.Dim, banner.Reset, banner.BrightGreen, banner.Reset)
		return nil
	},
}

// plugin list  —— 列出已安装的插件 (简洁表格)
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已安装的插件",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		plugins, err := mgr.Scan()
		if err != nil {
			return fmt.Errorf("获取插件列表失败: %w", err)
		}

		if len(plugins) == 0 {
			fmt.Println("暂无已安装的插件")
			fmt.Printf("插件目录: %s\n", mgr.PluginDir())
			return nil
		}

		fmt.Printf("%-5s %-20s %-10s %-8s %-20s %s\n",
			"", "名称", "版本", "状态", "入口", "描述")
		fmt.Println(strings.Repeat("─", 80))
		for _, p := range plugins {
			icon := p.Icon
			if icon == "" {
				icon = "🔌"
			}
			status := "启用"
			if !p.Enabled {
				status = "禁用"
			}
			entry := p.EntryPoint
			if entry == "" {
				entry = "-"
			}
			fmt.Printf("%-5s %-20s %-10s %-8s %-20s %s\n",
				icon, p.Name, p.Version, status, entry, p.Description)
		}
		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name]",
	Short: "安装/创建插件",
	Long: `在插件目录中创建插件骨架。

示例:
  jiasinecli plugin install MyTool        # 创建 plugin/MyTool/MyTool.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		logger.Info("正在安装插件: " + args[0])
		if err := mgr.Install(args[0]); err != nil {
			return fmt.Errorf("安装插件失败: %w", err)
		}
		fmt.Printf("%s✓ 插件 '%s' 安装成功%s\n", banner.BrightGreen, args[0], banner.Reset)
		fmt.Printf("  目录: %s/plugin/%s/\n", getExeDir(), args[0])
		fmt.Printf("  描述: plugin/%s/%s.json\n", args[0], args[0])
		fmt.Printf("\n  使用 %splugin %s%s 打开插件\n",
			banner.BrightGreen, args[0], banner.Reset)
		return nil
	},
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "卸载插件",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		if err := mgr.Remove(args[0]); err != nil {
			return fmt.Errorf("卸载插件失败: %w", err)
		}
		fmt.Printf("%s✓ 插件 '%s' 已卸载%s\n", banner.BrightGreen, args[0], banner.Reset)
		return nil
	},
}

func getExeDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

var pluginOpenCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "打开/运行插件",
	Long: `启动指定插件的入口程序。

等同于直接使用 plugin <name>。
插件的 entry_point 将在新的 cmd 窗口中启动。

示例:
  jiasinecli plugin open SerialTool`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		if err := mgr.Open(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s✓ 插件 '%s' 已启动%s\n", banner.BrightGreen, args[0], banner.Reset)
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginViewCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginOpenCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	rootCmd.AddCommand(pluginCmd)
}
