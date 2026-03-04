// Package service 提供远程服务调用管理
// 支持 HTTP/gRPC/进程 三种模式调用后端服务 (Python/C#/JS/TS/Java/Swift)
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Info 服务信息
type Info struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // http, grpc, process
	Address     string `json:"address"`     // 服务地址
	Status      string `json:"status"`      // running, stopped, unknown
	Description string `json:"description"`
}

// Manager 服务管理器
type Manager struct {
	services map[string]config.ServiceConfig
	client   *http.Client
}

// NewManager 创建服务管理器
func NewManager() *Manager {
	cfg := config.Get()
	return &Manager{
		services: cfg.Services,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// List 列出所有注册的服务
func (m *Manager) List() ([]Info, error) {
	var result []Info

	for name, svc := range m.services {
		info := Info{
			Name:        name,
			Type:        svc.Type,
			Address:     svc.Address,
			Description: svc.Description,
			Status:      "unknown",
		}

		// 快速探测状态
		if healthy, _ := m.healthCheckInternal(name); healthy {
			info.Status = "running"
		}

		result = append(result, info)
	}

	return result, nil
}

// Call 调用指定服务的方法
func (m *Manager) Call(serviceName, method string, params []string) (string, error) {
	svc, ok := m.services[serviceName]
	if !ok {
		return "", fmt.Errorf("服务 '%s' 未注册", serviceName)
	}

	switch svc.Type {
	case "http":
		return m.callHTTP(svc, method, params)
	case "process":
		return m.callProcess(svc, method, params)
	case "grpc":
		return m.callGRPC(svc, method, params)
	default:
		return "", fmt.Errorf("不支持的服务类型: %s", svc.Type)
	}
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck(serviceName string) (bool, error) {
	return m.healthCheckInternal(serviceName)
}

// callHTTP 通过 HTTP 调用服务
func (m *Manager) callHTTP(svc config.ServiceConfig, method string, params []string) (string, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(svc.Address, "/"), method)

	payload := map[string]interface{}{
		"method": method,
		"params": params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	logger.Debug("HTTP 调用", "url", url)

	resp, err := m.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("服务返回错误 [%d]: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// callProcess 通过子进程调用服务
func (m *Manager) callProcess(svc config.ServiceConfig, method string, params []string) (string, error) {
	args := append(svc.Args, method)
	args = append(args, params...)

	cmd := exec.Command(svc.Command, args...)
	if svc.WorkDir != "" {
		cmd.Dir = svc.WorkDir
	}

	// 设置环境变量
	for k, v := range svc.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	logger.Debug("进程调用", "command", svc.Command, "args", args)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("进程执行失败: %w\n输出: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// callGRPC gRPC 调用 (预留接口)
func (m *Manager) callGRPC(svc config.ServiceConfig, method string, params []string) (string, error) {
	// TODO: 实现 gRPC 调用
	// 可以使用 grpcurl 风格的动态调用或生成的客户端代码
	return "", fmt.Errorf("gRPC 调用尚未实现，服务: %s, 方法: %s", svc.Address, method)
}

// healthCheckInternal 内部健康检查
func (m *Manager) healthCheckInternal(serviceName string) (bool, error) {
	svc, ok := m.services[serviceName]
	if !ok {
		return false, fmt.Errorf("服务 '%s' 未注册", serviceName)
	}

	if svc.HealthCheck == "" {
		// 没有配置健康检查端点，尝试基本连通性
		if svc.Type == "http" {
			resp, err := m.client.Get(svc.Address)
			if err != nil {
				return false, nil
			}
			resp.Body.Close()
			return resp.StatusCode < 500, nil
		}
		return false, fmt.Errorf("服务 '%s' 未配置健康检查", serviceName)
	}

	url := fmt.Sprintf("%s/%s", strings.TrimRight(svc.Address, "/"), strings.TrimLeft(svc.HealthCheck, "/"))
	resp, err := m.client.Get(url)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
