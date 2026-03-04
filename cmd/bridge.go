package cmd

import (
	"fmt"

	"github.com/xiangjianhe-github/jiasinecli/internal/bridge"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"github.com/spf13/cobra"
)

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "桥接层管理 (FFI 动态库)",
	Long:  "管理通过 FFI 调用的原生动态库 (C/Rust/Objective-C/.NET Native AOT)",
}

var bridgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已加载的动态库",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := bridge.NewManager()
		libs, err := mgr.List()
		if err != nil {
			return fmt.Errorf("获取动态库列表失败: %w", err)
		}

		if len(libs) == 0 {
			fmt.Println("暂无已加载的动态库")
			return nil
		}

		fmt.Printf("%-20s %-10s %-10s %s\n", "名称", "类型", "平台", "路径")
		fmt.Println("--------------------------------------------------------------")
		for _, lib := range libs {
			fmt.Printf("%-20s %-10s %-10s %s\n", lib.Name, lib.Type, lib.Platform, lib.Path)
		}
		return nil
	},
}

var bridgeCallCmd = &cobra.Command{
	Use:   "call [library] [function] [args...]",
	Short: "调用动态库函数",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		libName := args[0]
		funcName := args[1]
		params := args[2:]

		logger.Info(fmt.Sprintf("调用动态库: %s.%s", libName, funcName))

		mgr := bridge.NewManager()
		result, err := mgr.Call(libName, funcName, params)
		if err != nil {
			return fmt.Errorf("调用动态库失败: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	bridgeCmd.AddCommand(bridgeListCmd)
	bridgeCmd.AddCommand(bridgeCallCmd)
	rootCmd.AddCommand(bridgeCmd)
}
