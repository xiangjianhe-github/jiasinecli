// Package config 提供配置管理
// 支持 YAML 配置文件 + 环境变量 + 命令行参数的优先级合并
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// AppConfig 应用配置结构
type AppConfig struct {
	// 日志配置
	Log LogConfig `yaml:"log" mapstructure:"log"`
	// 服务配置
	Services map[string]ServiceConfig `yaml:"services" mapstructure:"services"`
	// 桥接层配置
	Bridges map[string]BridgeConfig `yaml:"bridges" mapstructure:"bridges"`
	// 插件配置
	Plugins PluginConfig `yaml:"plugins" mapstructure:"plugins"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`   // debug, info, warn, error
	Format string `yaml:"format" mapstructure:"format"` // json, text
	Output string `yaml:"output" mapstructure:"output"` // stdout, file
	File   string `yaml:"file" mapstructure:"file"`     // 日志文件路径
}

// ServiceConfig 服务端配置
type ServiceConfig struct {
	Type        string            `yaml:"type" mapstructure:"type"`               // grpc, http, process
	Address     string            `yaml:"address" mapstructure:"address"`         // 服务地址
	Command     string            `yaml:"command" mapstructure:"command"`         // 启动命令 (process 类型)
	Args        []string          `yaml:"args" mapstructure:"args"`              // 命令参数
	WorkDir     string            `yaml:"workdir" mapstructure:"workdir"`         // 工作目录
	Env         map[string]string `yaml:"env" mapstructure:"env"`                // 环境变量
	HealthCheck string            `yaml:"health_check" mapstructure:"health_check"` // 健康检查端点
	Timeout     int               `yaml:"timeout" mapstructure:"timeout"`         // 超时(秒)
	Description string            `yaml:"description" mapstructure:"description"` // 描述
}

// BridgeConfig 动态库桥接配置
type BridgeConfig struct {
	Type     string            `yaml:"type" mapstructure:"type"`         // c, rust, dotnet
	Path     string            `yaml:"path" mapstructure:"path"`         // 动态库路径
	Platform map[string]string `yaml:"platform" mapstructure:"platform"` // 平台特定路径
	Functions []string          `yaml:"functions" mapstructure:"functions"` // 导出函数列表
}

// PluginConfig 插件配置
type PluginConfig struct {
	Dir     string `yaml:"dir" mapstructure:"dir"`         // 插件目录
	AutoLoad bool  `yaml:"auto_load" mapstructure:"auto_load"` // 自动加载
}

var cfg *AppConfig

// Init 初始化配置
func Init(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户目录失败: %w", err)
		}

		configDir := filepath.Join(home, ".jiasine")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 环境变量前缀
	viper.SetEnvPrefix("JIASINE")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值 - 这是正常的
	}

	cfg = &AppConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	return nil
}

// Get 获取当前配置
func Get() *AppConfig {
	if cfg == nil {
		cfg = &AppConfig{}
		setDefaults()
		viper.Unmarshal(cfg)
	}
	return cfg
}

func setDefaults() {
	home, _ := os.UserHomeDir()

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.file", filepath.Join(home, ".jiasine", "logs", "jiasine.log"))

	viper.SetDefault("plugins.dir", filepath.Join(home, ".jiasine", "plugins"))
	viper.SetDefault("plugins.auto_load", true)
}
