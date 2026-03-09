package cmd

import (
	"fmt"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/version"
	"github.com/spf13/cobra"
)

var (
	updateForce  bool // 强制更新，即使当前已是最新版本
	updateCheck  bool // 仅检查更新，不执行
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "检查并更新 JiasineCli 到最新版本",
	Long: `检查并更新 JiasineCli 到最新版本。

版本服务器: https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/version.json
二进制下载: https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/

示例:
  jiasinecli update              # 检查并更新
  jiasinecli update --check      # 仅检查，不执行更新
  jiasinecli update --force      # 强制重新下载当前版本`,
	RunE: func(cmd *cobra.Command, args []string) error {
		updater := version.NewUpdater()

		fmt.Printf("%s正在检查更新...%s\n", banner.Dim, banner.Reset)

		hasUpdate, remote, err := updater.CheckUpdate()
		if err != nil {
			return fmt.Errorf("检查更新失败: %w", err)
		}

		currentVer := version.Current
		currentVer.GitCommit = GitCommit
		currentVer.BuildDate = BuildDate

		if !hasUpdate && !updateForce {
			fmt.Printf("%s✓ 已是最新版本: %s%s\n", banner.BrightGreen, currentVer.String(), banner.Reset)
			if remote != nil {
				fmt.Printf("%s  发布日期: %s%s\n", banner.Dim, remote.ReleaseDate, banner.Reset)
			}
			return nil
		}

		if hasUpdate {
			fmt.Printf("\n%s发现新版本！%s\n", banner.BrightYellow, banner.Reset)
			fmt.Printf("  当前版本: %s%s%s\n", banner.Dim, currentVer.String(), banner.Reset)
			fmt.Printf("  最新版本: %s%s%s\n", banner.BrightGreen, remote.String(), banner.Reset)
			if remote.ReleaseDate != "" {
				fmt.Printf("  发布日期: %s\n", remote.ReleaseDate)
			}
			if remote.Changelog != "" {
				fmt.Printf("\n%s更新日志:%s\n%s\n", banner.BrightCyan, banner.Reset, remote.Changelog)
			}
		}

		// 仅检查模式
		if updateCheck {
			return nil
		}

		// 执行更新
		fmt.Println()
		startTime := time.Now()

		if err := updater.Update(remote); err != nil {
			return fmt.Errorf("更新失败: %w", err)
		}

		duration := time.Since(startTime)
		fmt.Printf("\n%s✓ 更新完成，耗时 %.1f 秒%s\n", banner.BrightGreen, duration.Seconds(), banner.Reset)
		fmt.Printf("%s请重新启动 JiasineCli 以使用新版本%s\n", banner.Dim, banner.Reset)

		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateCheck, "check", "c", false, "仅检查更新，不执行")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "强制重新下载当前版本")
	rootCmd.AddCommand(updateCmd)
}
