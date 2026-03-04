// Package ai 提供 AI 大模型统一调用接口
// 支持 ChatGPT、Claude、Gemini、Qwen、DeepSeek 等主流 AI 服务商
package ai

import (
	"fmt"
	"strings"
)

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message 聊天消息
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	WebSearch   bool      `json:"web_search,omitempty"` // 是否启用联网搜索
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	PromptTokens int   `json:"prompt_tokens"`
	OutputTokens int   `json:"output_tokens"`
	TotalTokens  int   `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
	Error   error  `json:"-"`
}

// Provider AI 服务提供商接口
type Provider interface {
	// Name 提供商名称
	Name() string
	// Chat 发送聊天请求（非流式）
	Chat(req *ChatRequest) (*ChatResponse, error)
	// ChatStream 发送聊天请求（流式）
	ChatStream(req *ChatRequest) (<-chan StreamChunk, error)
	// Models 列出可用模型
	Models() []string
	// DefaultModel 默认模型
	DefaultModel() string
}

// ProviderConfig 服务商配置
type ProviderConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`       // 提供商标识: openai, claude, gemini, qwen, deepseek
	APIKey  string `yaml:"api_key" mapstructure:"api_key"` // API 密钥
	BaseURL string `yaml:"base_url" mapstructure:"base_url"` // 自定义 API 地址（可选）
	Model   string `yaml:"model" mapstructure:"model"`     // 默认模型（可选）
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"` // 是否启用
}

// ProviderFactory 提供商工厂函数
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

// 全局注册表
var providerFactories = map[string]ProviderFactory{}

// RegisterProvider 注册提供商工厂
func RegisterProvider(name string, factory ProviderFactory) {
	providerFactories[strings.ToLower(name)] = factory
}

// NewProvider 根据配置创建提供商实例
func NewProvider(cfg ProviderConfig) (Provider, error) {
	factory, ok := providerFactories[strings.ToLower(cfg.Name)]
	if !ok {
		return nil, fmt.Errorf("未知的 AI 提供商: %s (支持: openai, claude, gemini, qwen, deepseek)", cfg.Name)
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("AI 提供商 '%s' 未配置 API Key", cfg.Name)
	}
	return factory(cfg)
}

// SupportedProviders 返回已注册的提供商列表
func SupportedProviders() []string {
	names := make([]string, 0, len(providerFactories))
	for name := range providerFactories {
		names = append(names, name)
	}
	return names
}
