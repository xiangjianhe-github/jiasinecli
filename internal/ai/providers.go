package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	// 流式暂用非流式替代
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
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
		cfg.BaseURL = "https://api.anthropic.com/v1"
	}
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
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}

	// Claude API: system 在顶层，messages 只有 user/assistant
	var system string
	var messages []map[string]string
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
		} else {
			messages = append(messages, map[string]string{
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

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", p.cfg.BaseURL+"/messages", bytes.NewReader(data))
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
	for _, block := range result.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "Claude",
		PromptTokens: result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens,
	}, nil
}

func (p *claudeProvider) ChatStream(req *ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

type claudeChatResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
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
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
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
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
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
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Chat(req)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}
