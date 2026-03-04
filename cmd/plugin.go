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

插件市场优先从远程服务器获取列表，服务器不可达时回退到本地。
每个插件 = <Name>.json 描述文件 + <Name>.7z 压缩包。
安装时自动下载并解压到 plugin/<Name>/ 目录。

示例:
  jiasinecli plugin view                 # 查看插件市场 (远程优先)
  jiasinecli plugin open SerialTool      # 打开已安装的 SerialTool 插件
  jiasinecli plugin list                 # 列出已安装的插件
  jiasinecli plugin install SerialTool   # 安装插件 (下载+解压)
  jiasinecli plugin remove MyPlugin      # 卸载插件`,
}

// plugin view  —— 插件市场：远程优先 + 本地回退
var pluginViewCmd = &cobra.Command{
	Use:   "view",
	Short: "查看插件市场 (远程优先，本地回退)",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()

		fmt.Printf("\n%s%s 插件市场%s  正在获取列表...\n",
			banner.Bold+banner.BrightCyan, "📦", banner.Reset)

		plugins, source, err := mgr.Marketplace()
		if err != nil {
			return fmt.Errorf("获取插件市场失败: %w", err)
		}

		// 来源指示
		sourceLabel := "🌐 远程服务器"
		if source == plugin.SourceLocal {
			sourceLabel = "📁 本地目录"
		}

		fmt.Printf("\r%s%s 插件市场%s  %s来源: %s%s\n",
			banner.Bold+banner.BrightCyan, "📦", banner.Reset,
			banner.Dim, sourceLabel, banner.Reset)
		fmt.Println(strings.Repeat("─", 70))

		if len(plugins) == 0 {
			fmt.Printf("\n  %s暂无可用插件%s\n", banner.Dim, banner.Reset)
			if source == plugin.SourceLocal {
				fmt.Printf("  将插件 .json 文件放入 %s%s%s 即可被发现\n",
					banner.BrightGreen, mgr.PluginDir(), banner.Reset)
				fmt.Printf("  格式: plugin/<名称>.json + plugin/<名称>.7z\n\n")
			}
			return nil
		}

		for i, p := range plugins {
			icon := p.Icon
			if icon == "" {
				icon = "🔌"
			}

			// 状态
			status := fmt.Sprintf("%s可用%s", banner.BrightGreen, banner.Reset)
			if p.Installed {
				status = fmt.Sprintf("%s✓ 已安装%s", banner.BrightGreen, banner.Reset)
			}
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

		fmt.Printf("\n%s提示%s: 使用 %splugin install <名称>%s 安装，%splugin open <名称>%s 打开\n\n",
			banner.Dim, banner.Reset,
			banner.BrightGreen, banner.Reset,
			banner.BrightGreen, banner.Reset)
		return nil
	},
}

// plugin list  —— 列出已安装的插件
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已安装的插件",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		plugins, err := mgr.ScanInstalled()
		if err != nil {
			return fmt.Errorf("获取已安装插件列表失败: %w", err)
		}

		if len(plugins) == 0 {
			fmt.Println("暂无已安装的插件")
			fmt.Printf("使用 %splugin view%s 查看可用插件，%splugin install <名称>%s 安装\n",
				banner.BrightGreen, banner.Reset,
				banner.BrightGreen, banner.Reset)
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
	Short: "安装插件 (从远程下载或本地安装)",
	Long: `从插件市场安装指定插件。

安装流程:
  1. 从远程服务器下载 <Name>.json + <Name>.7z
  2. 若远程不可达，使用本地 plugin/ 目录下的文件
  3. 解压 .7z 到 plugin/<Name>/ 目录

前置条件:
  需要安装 7-Zip (https://www.7-zip.org/)

示例:
  jiasinecli plugin install SerialTool`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		name := args[0]

		fmt.Printf("正在安装插件 '%s'...\n", name)
		logger.Info("正在安装插件: " + name)

		if err := mgr.Install(name); err != nil {
			return fmt.Errorf("安装插件失败: %w", err)
		}

		fmt.Printf("%s✓ 插件 '%s' 安装成功%s\n", banner.BrightGreen, name, banner.Reset)
		fmt.Printf("  目录: %s/plugin/%s/\n", getExeDir(), name)
		fmt.Printf("\n  使用 %splugin open %s%s 打开插件\n",
			banner.BrightGreen, name, banner.Reset)
		return nil
	},
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "卸载已安装的插件",
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
	Short: "打开/运行已安装的插件",
	Long: `启动指定插件的入口程序。

插件的 entry_point 将在新的 cmd 窗口中启动。
插件必须已安装 (使用 plugin install 安装)。

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
