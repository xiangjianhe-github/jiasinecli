package cmd

import (
	"fmt"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"github.com/xiangjianhe-github/jiasinecli/internal/plugin"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "插件管理",
	Long:  "管理 Jiasine CLI 插件 (列表、安装、卸载、启用/禁用)",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已安装的插件",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		plugins, err := mgr.List()
		if err != nil {
			return fmt.Errorf("获取插件列表失败: %w", err)
		}

		if len(plugins) == 0 {
			fmt.Println("暂无已安装的插件")
			return nil
		}

		fmt.Printf("%-20s %-10s %-10s %s\n", "名称", "版本", "状态", "描述")
		fmt.Println("--------------------------------------------------------------")
		for _, p := range plugins {
			status := "启用"
			if !p.Enabled {
				status = "禁用"
			}
			fmt.Printf("%-20s %-10s %-10s %s\n", p.Name, p.Version, status, p.Description)
		}
		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [name]",
	Short: "安装插件",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.NewManager()
		logger.Info("正在安装插件: " + args[0])
		if err := mgr.Install(args[0]); err != nil {
			return fmt.Errorf("安装插件失败: %w", err)
		}
		fmt.Printf("插件 '%s' 安装成功\n", args[0])
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
		fmt.Printf("插件 '%s' 已卸载\n", args[0])
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	rootCmd.AddCommand(pluginCmd)
}
