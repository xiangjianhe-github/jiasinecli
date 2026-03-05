// Package config 提供配置管理
// 支持 YAML 配置文件 + 环境变量 + 命令行参数的优先级合并
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/ai"
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
	// AI 配置
	AI AIConfig `yaml:"ai" mapstructure:"ai"`
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
	Dir      string `yaml:"dir" mapstructure:"dir"`           // 插件目录
	AutoLoad bool   `yaml:"auto_load" mapstructure:"auto_load"` // 自动加载
}

// AIConfig AI 配置
type AIConfig struct {
	Active    string                       `yaml:"active" mapstructure:"active"`           // 当前使用的提供商
	WebSearch bool                         `yaml:"web_search" mapstructure:"web_search"` // 默认启用联网搜索
	Providers map[string]ai.ProviderConfig `yaml:"providers" mapstructure:"providers"`     // 各提供商配置
	Agents    ai.AgentConfig               `yaml:"agents" mapstructure:"agents"`           // Agent 配置
	Skills    ai.SkillConfig               `yaml:"skills" mapstructure:"skills"`           // Skill 配置
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

// Reload 重新加载配置文件
func Reload() error {
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	cfg = &AppConfig{}
	return viper.Unmarshal(cfg)
}

// SetActiveProvider 切换当前激活的 AI 提供商并持久化到配置文件
func SetActiveProvider(name string) error {
	viper.Set("ai.active", name)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	// 更新内存中的配置
	if cfg != nil {
		cfg.AI.Active = name
	}
	return nil
}

// SetWebSearch 切换联网搜索设置并持久化到配置文件
func SetWebSearch(enabled bool) error {
	viper.Set("ai.web_search", enabled)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	// 更新内存中的配置
	if cfg != nil {
		cfg.AI.WebSearch = enabled
	}
	return nil
}

// EnsureAIConfig 检查配置文件是否存在并包含 AI 配置
// 如果不存在，自动生成模板文件，返回文件路径和是否新建
func EnsureAIConfig() (configPath string, created bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("获取用户目录失败: %w", err)
	}

	configDir := filepath.Join(home, ".jiasine")
	configPath = filepath.Join(configDir, "config.yaml")

	// 确保目录存在
	os.MkdirAll(configDir, 0755)

	// 检查文件是否存在
	if _, statErr := os.Stat(configPath); statErr == nil {
		// 文件存在，检查是否包含 ai: 配置段
		data, readErr := os.ReadFile(configPath)
		if readErr == nil {
			content := string(data)
			// 文件中已有 ai: 段 → 不需要再生成模板
			if strings.Contains(content, "\nai:") || strings.HasPrefix(content, "ai:") {
				return configPath, false, nil
			}
		}
	}

	// 不存在或无有效 AI 配置 — 生成模板
	tmpl := generateAIConfigTemplate()

	// 如果文件已存在（但缺 AI 配置），追加；否则新建
	if _, statErr := os.Stat(configPath); statErr == nil {
		// 读取现有内容检查是否已有 ai: 段
		data, _ := os.ReadFile(configPath)
		if !strings.Contains(string(data), "\nai:") {
			f, appendErr := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
			if appendErr != nil {
				return configPath, false, appendErr
			}
			defer f.Close()
			f.WriteString("\n" + tmpl)
		}
		return configPath, true, nil
	}

	// 全新文件
	writeErr := os.WriteFile(configPath, []byte(tmpl), 0644)
	if writeErr != nil {
		return configPath, false, writeErr
	}
	return configPath, true, nil
}

func generateAIConfigTemplate() string {
	return `# Jiasine CLI 配置文件
# 请填入你的 AI 服务商 API Key

ai:
  active: deepseek           # 当前使用的服务商 (修改为你要用的)
  web_search: false          # 默认启用联网搜索 (true/false)

  providers:
    # DeepSeek (推荐入门，价格低)
    # 获取 API Key: https://platform.deepseek.com
    deepseek:
      name: deepseek
      api_key: ""            # ← 填入你的 DeepSeek API Key
      model: "deepseek-chat"
      enabled: true

    # OpenAI (ChatGPT)
    # 获取 API Key: https://platform.openai.com/api-keys
    openai:
      name: openai
      api_key: ""            # ← 填入你的 OpenAI API Key
      base_url: "https://api.openai.com/v1"
      model: "gpt-4o"
      enabled: false         # 改为 true 启用

    # Anthropic (Claude)
    # 获取 API Key: https://console.anthropic.com
    claude:
      name: claude
      api_key: ""            # ← 填入你的 Anthropic API Key
      model: "claude-sonnet-4-20250514"
      enabled: false

    # Google (Gemini)
    # 获取 API Key: https://aistudio.google.com/apikey
    gemini:
      name: gemini
      api_key: ""            # ← 填入你的 Google AI API Key
      model: "gemini-2.5-pro"
      enabled: false

    # 阿里云 通义千问 (Qwen)
    # 获取 API Key: https://dashscope.console.aliyun.com
    qwen:
      name: qwen
      api_key: ""            # ← 填入你的通义千问 API Key
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      model: "qwen-max"
      enabled: false

  agents:
    dir: ""                  # Agent 定义目录 (默认 ~/.jiasine/agents)

  skills:
    dir: ""                  # Skills 定义目录 (默认 ~/.jiasine/skills)
`
}

// HasValidAIProviders 检查当前配置是否包含有效的 AI 提供商
func HasValidAIProviders() bool {
	c := Get()
	for _, p := range c.AI.Providers {
		if p.Enabled && p.APIKey != "" {
			return true
		}
	}
	return false
}

func setDefaults() {
	home, _ := os.UserHomeDir()

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.file", filepath.Join(home, ".jiasine", "logs", "jiasine.log"))

	viper.SetDefault("plugins.dir", filepath.Join(home, ".jiasine", "plugins"))
	viper.SetDefault("plugins.auto_load", true)

	// AI 默认值
	viper.SetDefault("ai.active", "")
	viper.SetDefault("ai.agents.dir", filepath.Join(home, ".jiasine", "agents"))
	viper.SetDefault("ai.skills.dir", filepath.Join(home, ".jiasine", "skills"))
}
