# Jiasine CLI - Makefile
# 支持跨平台编译: Windows, macOS, Linux, Raspberry Pi

APP_NAME := jiasinecli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Go 构建参数
LDFLAGS := -ldflags "-s -w \
	-X github.com/xiangjianhe-github/jiasinecli/cmd.Version=$(VERSION) \
	-X github.com/xiangjianhe-github/jiasinecli/cmd.GitCommit=$(GIT_COMMIT) \
	-X github.com/xiangjianhe-github/jiasinecli/cmd.BuildDate=$(BUILD_DATE)"

# 输出目录
DIST_DIR := dist

.PHONY: all build clean test lint dev install cross cross-windows cross-darwin cross-linux cross-raspi help

## 默认目标
all: clean test build

## 开发构建 (当前平台)
dev:
	go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME) .

## 生产构建 (当前平台, 优化)
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME) .

## 安装到 GOPATH/bin
install:
	CGO_ENABLED=0 go install $(LDFLAGS) .

## 运行测试
test:
	go test ./... -v -cover

## 代码检查
lint:
	golangci-lint run ./...

## 清理构建产物
clean:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)

## ============ 跨平台编译 ============

## 编译所有平台 (7 targets)
cross: cross-windows cross-darwin cross-linux cross-raspi
	@echo "所有平台编译完成！(7 targets)"
	@ls -la $(DIST_DIR)/

## Windows (amd64, arm64)
cross-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe .

## macOS (Intel amd64, Apple Silicon arm64)
cross-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 .

## Linux (amd64, arm64)
cross-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 .

## Raspberry Pi (Linux ARMv7)
cross-raspi:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-raspi .

## 帮助
help:
	@echo "可用目标:"
	@echo "  make dev           - 开发构建"
	@echo "  make build         - 生产构建"
	@echo "  make test          - 运行测试"
	@echo "  make cross         - 跨平台编译 (所有 7 平台)"
	@echo "  make cross-windows - 编译 Windows 版本 (amd64, arm64)"
	@echo "  make cross-darwin  - 编译 macOS 版本 (Intel, Apple Silicon)"
	@echo "  make cross-linux   - 编译 Linux 版本 (amd64, arm64)"
	@echo "  make cross-raspi   - 编译 Raspberry Pi 版本 (ARMv7)"
	@echo "  make install       - 安装到 GOPATH"
	@echo "  make clean         - 清理构建产物"
