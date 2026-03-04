package cmd

import (
	"fmt"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
	"github.com/xiangjianhe-github/jiasinecli/internal/service"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "服务管理",
	Long:  "管理后端独立服务 (Python/C#/JS/TS/Java/Swift 服务的启动、停止、状态查询)",
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有已注册的服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.NewManager()
		services, err := mgr.List()
		if err != nil {
			return fmt.Errorf("获取服务列表失败: %w", err)
		}

		if len(services) == 0 {
			fmt.Println("暂无已注册的服务")
			return nil
		}

		fmt.Printf("%-20s %-10s %-15s %-10s %s\n", "名称", "类型", "地址", "状态", "描述")
		fmt.Println("------------------------------------------------------------------------")
		for _, s := range services {
			fmt.Printf("%-20s %-10s %-15s %-10s %s\n", s.Name, s.Type, s.Address, s.Status, s.Description)
		}
		return nil
	},
}

var serviceCallCmd = &cobra.Command{
	Use:   "call [service] [method] [args...]",
	Short: "调用远程服务方法",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svcName := args[0]
		method := args[1]
		params := args[2:]

		logger.Info(fmt.Sprintf("调用服务: %s.%s", svcName, method))

		mgr := service.NewManager()
		result, err := mgr.Call(svcName, method, params)
		if err != nil {
			return fmt.Errorf("调用服务失败: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

var serviceHealthCmd = &cobra.Command{
	Use:   "health [service]",
	Short: "检查服务健康状态",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.NewManager()
		healthy, err := mgr.HealthCheck(args[0])
		if err != nil {
			return fmt.Errorf("健康检查失败: %w", err)
		}

		if healthy {
			fmt.Printf("服务 '%s' 运行正常 ✓\n", args[0])
		} else {
			fmt.Printf("服务 '%s' 不可用 ✗\n", args[0])
		}
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceCallCmd)
	serviceCmd.AddCommand(serviceHealthCmd)
	rootCmd.AddCommand(serviceCmd)
}
