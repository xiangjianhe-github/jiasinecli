// Package icon 嵌入 icon.json (Lottie 动画定义) 供程序使用
// icon.json 由 icon.json 文件定义，是一个标准的 Lottie 动画
// 可用于 Web UI 渲染状态栏图标
package icon

import (
	_ "embed"
)

//go:embed icon.json
var LottieJSON []byte

// GetLottieJSON 返回 Lottie 动画 JSON 数据
func GetLottieJSON() []byte {
	return LottieJSON
}
