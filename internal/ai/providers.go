package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ============================================================
// OpenAI (ChatGPT) 提供商
// ============================================================

type openaiProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

func init() {
	// OpenAI / ChatGPT
	RegisterProvider("openai", newOpenAI)
	RegisterProvider("chatgpt", newOpenAI)
	// Claude / Anthropic
	RegisterProvider("claude", newClaude)
	RegisterProvider("anthropic", newClaude)
	// Gemini / Google
	RegisterProvider("gemini", newGemini)
	RegisterProvider("google", newGemini)
	// Qwen / 通义千问
	RegisterProvider("qwen", newQwen)
	RegisterProvider("tongyi", newQwen)
	// DeepSeek
	RegisterProvider("deepseek", newDeepSeek)
}

func newOpenAI(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	return &openaiProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *openaiProvider) Name() string { return "OpenAI" }

func (p *openaiProvider) Models() []string {
	return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo", "o1", "o1-mini", "o3-mini"}
}

func (p *openaiProvider) DefaultModel() string {
	if p.cfg.Model != "" {
		return p.cfg.Model
	}
	return "gpt-4o"
}

func (p *openaiProvider) Chat(req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	// OpenAI: 启用联网搜索 (web_search tool)
	if req.WebSearch {
		body["tools"] = []map[string]interface{}{
			{"type": "web_search_preview"},
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenAI 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 OpenAI 响应失败: %w", err)
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "OpenAI",
		PromptTokens: result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		TotalTokens:  result.Usage.TotalTokens,
	}, nil
}

func (p *openaiProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.WebSearch {
		body["tools"] = []map[string]interface{}{
			{"type": "web_search_preview"},
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	// 流式请求不设超时，用不带 timeout 的 client
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 OpenAI 失败: %w", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenAI API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseOpenAISSE(resp.Body, "OpenAI", model, ch)
	}()
	return ch, nil
}

type openaiChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ============================================================
// Claude (Anthropic) 提供商
// ============================================================

type claudeProvider struct {
	cfg    ProviderConfig
	client *http.Client
}



func newClaude(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &claudeProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *claudeProvider) Name() string { return "Claude" }

func (p *claudeProvider) Models() []string {
	return []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-3.5-sonnet-20241022", "claude-3-haiku-20240307"}
}

func (p *claudeProvider) DefaultModel() string {
	if p.cfg.Model != "" {
		return p.cfg.Model
	}
	return "claude-sonnet-4-20250514"
}

func (p *claudeProvider) Chat(req *ChatRequest) (*ChatResponse, error) {
	return p.chatNative(req)
}

// chatNative 使用 Anthropic 原生 Messages API
func (p *claudeProvider) chatNative(req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	// Claude API: system 在顶层，messages 只有 user/assistant
	var system string
	var messages []interface{}
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
		} else if m.Role == RoleToolResult {
			// 工具返回结果 — 作为 user 角色发送
			var toolResults []map[string]interface{}
			if err := json.Unmarshal([]byte(m.Content), &toolResults); err == nil {
				messages = append(messages, map[string]interface{}{
					"role":    "user",
					"content": toolResults,
				})
			}
		} else if m.Role == RoleAssistantToolUse {
			// 助手的工具调用内容块 — 原样传回
			var contentBlocks []interface{}
			if err := json.Unmarshal([]byte(m.Content), &contentBlocks); err == nil {
				messages = append(messages, map[string]interface{}{
					"role":    "assistant",
					"content": contentBlocks,
				})
			}
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    string(m.Role),
				"content": m.Content,
			})
		}
	}

	body := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": 4096,
	}
	if system != "" {
		body["system"] = system
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	// 合并工具定义: MCP tools + web_search
	var allTools []map[string]interface{}
	if len(req.Tools) > 0 {
		allTools = append(allTools, req.Tools...)
	}
	if req.WebSearch {
		allTools = append(allTools, map[string]interface{}{
			"type": "web_search_20250305",
			"name": "web_search",
		})
	}
	if len(allTools) > 0 {
		body["tools"] = allTools
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 智能拼接 URL:
	//   已含 /messages       → 直接用 (如 https://proxy.com/v1/messages)
	//   已含 /v1             → 加 /messages (如 https://api.anthropic.com/v1)
	//   仅域名               → 加 /v1/messages (如 https://proxy.com)
	apiURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if strings.HasSuffix(apiURL, "/messages") {
		// 已包含完整路径，直接用
	} else if strings.HasSuffix(apiURL, "/v1") {
		apiURL += "/messages"
	} else {
		apiURL += "/v1/messages"
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Claude 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Claude API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var result claudeChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 Claude 响应失败: %w", err)
	}

	content := ""
	var toolCalls []ToolCall
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	chatResp := &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "Claude",
		PromptTokens: result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens,
		ToolCalls:    toolCalls,
		StopReason:   result.StopReason,
	}

	return chatResp, nil
}

func (p *claudeProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	// 构建请求体 (与 chatNative 相同，但加 stream: true)
	var system string
	var messages []interface{}
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
		} else if m.Role == RoleToolResult {
			var toolResults []map[string]interface{}
			if err := json.Unmarshal([]byte(m.Content), &toolResults); err == nil {
				messages = append(messages, map[string]interface{}{
					"role":    "user",
					"content": toolResults,
				})
			}
		} else if m.Role == RoleAssistantToolUse {
			var contentBlocks []interface{}
			if err := json.Unmarshal([]byte(m.Content), &contentBlocks); err == nil {
				messages = append(messages, map[string]interface{}{
					"role":    "assistant",
					"content": contentBlocks,
				})
			}
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    string(m.Role),
				"content": m.Content,
			})
		}
	}

	body := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": 4096,
		"stream":     true,
	}
	if system != "" {
		body["system"] = system
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	// 合并工具定义
	var allTools []map[string]interface{}
	if len(req.Tools) > 0 {
		allTools = append(allTools, req.Tools...)
	}
	if req.WebSearch {
		allTools = append(allTools, map[string]interface{}{
			"type": "web_search_20250305",
			"name": "web_search",
		})
	}
	if len(allTools) > 0 {
		body["tools"] = allTools
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if strings.HasSuffix(apiURL, "/messages") {
		// 已包含完整路径
	} else if strings.HasSuffix(apiURL, "/v1") {
		apiURL += "/messages"
	} else {
		apiURL += "/v1/messages"
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Claude 失败: %w", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Claude API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseClaudeSSE(resp.Body, model, ch)
	}()
	return ch, nil
}

type claudeChatResponse struct {
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type  string                 `json:"type"`
		Text  string                 `json:"text,omitempty"`
		ID    string                 `json:"id,omitempty"`
		Name  string                 `json:"name,omitempty"`
		Input map[string]interface{} `json:"input,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ============================================================
// Gemini (Google) 提供商
// ============================================================

type geminiProvider struct {
	cfg    ProviderConfig
	client *http.Client
}



func newGemini(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &geminiProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *geminiProvider) Name() string { return "Gemini" }

func (p *geminiProvider) Models() []string {
	return []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash", "gemini-1.5-pro"}
}

func (p *geminiProvider) DefaultModel() string {
	if p.cfg.Model != "" {
		return p.cfg.Model
	}
	return "gemini-2.5-flash"
}

func (p *geminiProvider) Chat(req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	// Gemini API 格式
	var parts []map[string]string
	for _, m := range req.Messages {
		parts = append(parts, map[string]string{"text": m.Content})
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": parts},
		},
	}
	if req.Temperature > 0 {
		body["generationConfig"] = map[string]interface{}{
			"temperature": req.Temperature,
		}
	}
	// Gemini: 启用联网搜索 (Google Search grounding)
	if req.WebSearch {
		body["tools"] = []map[string]interface{}{
			{"google_search": map[string]interface{}{}},
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.cfg.BaseURL, model, p.cfg.APIKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Gemini 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var result geminiChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 Gemini 响应失败: %w", err)
	}

	content := ""
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		content = result.Candidates[0].Content.Parts[0].Text
	}

	return &ChatResponse{
		Content:      content,
		Model:        model,
		Provider:     "Gemini",
		PromptTokens: result.UsageMetadata.PromptTokenCount,
		OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  result.UsageMetadata.TotalTokenCount,
	}, nil
}

func (p *geminiProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	// Gemini SSE 格式不同，暂用非流式模拟
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Type: "error", Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Type: "content", Content: resp.Content, Model: resp.Model, Provider: resp.Provider}
		ch <- StreamChunk{
			Type: "usage", Done: true,
			Model: resp.Model, Provider: resp.Provider,
			Usage: &TokenUsage{
				PromptTokens: resp.PromptTokens,
				OutputTokens: resp.OutputTokens,
				TotalTokens:  resp.TotalTokens,
			},
		}
	}()
	return ch, nil
}

type geminiChatResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// ============================================================
// Qwen (通义千问 / 阿里云) 提供商
// 兼容 OpenAI API 格式
// ============================================================

type qwenProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

func newQwen(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	return &qwenProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *qwenProvider) Name() string { return "Qwen" }

func (p *qwenProvider) Models() []string {
	return []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long", "qwen-vl-max"}
}

func (p *qwenProvider) DefaultModel() string {
	if p.cfg.Model != "" {
		return p.cfg.Model
	}
	return "qwen-plus"
}

func (p *qwenProvider) Chat(req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	// Qwen: 启用联网搜索
	if req.WebSearch {
		body["enable_search"] = true
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Qwen 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Qwen API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	// Qwen 兼容 OpenAI 响应格式
	var result openaiChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 Qwen 响应失败: %w", err)
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "Qwen",
		PromptTokens: result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		TotalTokens:  result.Usage.TotalTokens,
	}, nil
}

func (p *qwenProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.WebSearch {
		body["enable_search"] = true
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 Qwen 失败: %w", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Qwen API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseOpenAISSE(resp.Body, "Qwen", model, ch)
	}()
	return ch, nil
}

// ============================================================
// DeepSeek 提供商
// 兼容 OpenAI API 格式
// ============================================================

type deepSeekProvider struct {
	cfg    ProviderConfig
	client *http.Client
}

func newDeepSeek(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com/v1"
	}
	return &deepSeekProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *deepSeekProvider) Name() string { return "DeepSeek" }

func (p *deepSeekProvider) Models() []string {
	return []string{"deepseek-chat", "deepseek-coder", "deepseek-reasoner"}
}

func (p *deepSeekProvider) DefaultModel() string {
	if p.cfg.Model != "" {
		return p.cfg.Model
	}
	return "deepseek-chat"
}

func (p *deepSeekProvider) Chat(req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	// DeepSeek 不支持 web_search 工具类型，跳过联网搜索

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 DeepSeek 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DeepSeek API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 DeepSeek 响应失败: %w", err)
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "DeepSeek",
		PromptTokens: result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		TotalTokens:  result.Usage.TotalTokens,
	}, nil
}

func (p *deepSeekProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	// DeepSeek 不支持 web_search 工具类型，跳过联网搜索

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 DeepSeek 失败: %w", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("DeepSeek API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseOpenAISSE(resp.Body, "DeepSeek", model, ch)
	}()
	return ch, nil
}

// ============================================================
// SSE 解析器
// ============================================================

// parseOpenAISSE 解析 OpenAI 兼容格式的 SSE 流 (OpenAI/DeepSeek/Qwen)
// DeepSeek R1 模型额外支持 reasoning_content 字段
func parseOpenAISSE(body io.Reader, provider, model string, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var usage *TokenUsage

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: "data: {...}" 或 "data: [DONE]"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// 解析 JSON chunk
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"` // DeepSeek R1
					Role             string `json:"role"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Model != "" {
			model = chunk.Model
		}

		// 提取 usage (通常在最后一个 chunk)
		if chunk.Usage != nil {
			usage = &TokenUsage{
				PromptTokens: chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// DeepSeek R1: 思考/推理内容
		if delta.ReasoningContent != "" {
			ch <- StreamChunk{
				Type:     "thinking",
				Thinking: delta.ReasoningContent,
				Model:    model,
				Provider: provider,
			}
		}

		// 正文内容
		if delta.Content != "" {
			ch <- StreamChunk{
				Type:     "content",
				Content:  delta.Content,
				Model:    model,
				Provider: provider,
			}
		}
	}

	// 发送最终的 done 信号
	ch <- StreamChunk{
		Type:     "usage",
		Done:     true,
		Model:    model,
		Provider: provider,
		Usage:    usage,
	}
}

// parseClaudeSSE 解析 Claude (Anthropic) SSE 流
// 支持 thinking (扩展思考) 、text、tool_use 三种内容块
func parseClaudeSSE(body io.Reader, model string, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var stopReason string
	var inputTokens, outputTokens int

	// 跟踪当前内容块类型和工具调用
	type blockInfo struct {
		blockType string // "thinking", "text", "tool_use"
		toolID    string
		toolName  string
		inputJSON strings.Builder
	}
	blocks := map[int]*blockInfo{}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// 解析事件
		var event struct {
			Type    string `json:"type"`
			Message *struct {
				Model string `json:"model"`
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message,omitempty"`
			Index        int `json:"index"`
			ContentBlock *struct {
				Type string `json:"type"` // "thinking", "text", "tool_use"
				ID   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"content_block,omitempty"`
			Delta *struct {
				Type             string `json:"type"` // "thinking_delta", "text_delta", "input_json_delta"
				Thinking         string `json:"thinking,omitempty"`
				Text             string `json:"text,omitempty"`
				PartialJSON      string `json:"partial_json,omitempty"`
				StopReason       string `json:"stop_reason,omitempty"`
			} `json:"delta,omitempty"`
			Usage *struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				if event.Message.Model != "" {
					model = event.Message.Model
				}
				inputTokens = event.Message.Usage.InputTokens
			}

		case "content_block_start":
			if event.ContentBlock != nil {
				blocks[event.Index] = &blockInfo{
					blockType: event.ContentBlock.Type,
					toolID:    event.ContentBlock.ID,
					toolName:  event.ContentBlock.Name,
				}
			}

		case "content_block_delta":
			if event.Delta == nil {
				continue
			}
			switch event.Delta.Type {
			case "thinking_delta":
				if event.Delta.Thinking != "" {
					ch <- StreamChunk{
						Type:     "thinking",
						Thinking: event.Delta.Thinking,
						Model:    model,
						Provider: "Claude",
					}
				}
			case "text_delta":
				if event.Delta.Text != "" {
					ch <- StreamChunk{
						Type:     "content",
						Content:  event.Delta.Text,
						Model:    model,
						Provider: "Claude",
					}
				}
			case "input_json_delta":
				// 工具调用的 JSON 输入持续拼接
				if bi, ok := blocks[event.Index]; ok {
					bi.inputJSON.WriteString(event.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			if bi, ok := blocks[event.Index]; ok && bi.blockType == "tool_use" {
				// 工具调用完成，解析输入 JSON
				var input map[string]interface{}
				_ = json.Unmarshal([]byte(bi.inputJSON.String()), &input)
				ch <- StreamChunk{
					Type:     "tool_use",
					Model:    model,
					Provider: "Claude",
					ToolCalls: []ToolCall{{
						ID:    bi.toolID,
						Name:  bi.toolName,
						Input: input,
					}},
				}
			}

		case "message_delta":
			if event.Delta != nil && event.Delta.StopReason != "" {
				stopReason = event.Delta.StopReason
			}
			if event.Usage != nil {
				outputTokens = event.Usage.OutputTokens
			}

		case "message_stop":
			// 流结束
		}
	}

	// 发送 done 信号
	ch <- StreamChunk{
		Type:       "usage",
		Done:       true,
		Model:      model,
		Provider:   "Claude",
		StopReason: stopReason,
		Usage: &TokenUsage{
			PromptTokens: inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		},
	}
}
