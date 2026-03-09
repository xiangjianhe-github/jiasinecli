package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// 版本配置服务器地址
	versionServerURL = "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/version.json"
	// 二进制文件下载地址前缀
	binaryDownloadPrefix = "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/"
)

// RemoteVersion 远程版本信息
type RemoteVersion struct {
	Info
	DownloadURL map[string]string `json:"download_urls"` // platform -> URL
	ReleaseDate string            `json:"release_date"`
	Changelog   string            `json:"changelog"`
}

// Updater 自动更新管理器
type Updater struct {
	currentVersion Info
	httpClient     *http.Client
}

// NewUpdater 创建更新管理器
func NewUpdater() *Updater {
	return &Updater{
		currentVersion: Current,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckUpdate 检查是否有新版本
// 返回: (有更新, 远程版本信息, 错误)
func (u *Updater) CheckUpdate() (bool, *RemoteVersion, error) {
	// 获取远程版本信息
	remote, err := u.fetchRemoteVersion()
	if err != nil {
		return false, nil, fmt.Errorf("获取远程版本失败: %w", err)
	}

	// 比较版本
	if remote.Compare(u.currentVersion) > 0 {
		return true, remote, nil
	}

	return false, remote, nil
}

// fetchRemoteVersion 从服务器获取版本信息
func (u *Updater) fetchRemoteVersion() (*RemoteVersion, error) {
	resp, err := u.httpClient.Get(versionServerURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误: %d", resp.StatusCode)
	}

	var remote RemoteVersion
	if err := json.NewDecoder(resp.Body).Decode(&remote); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}

	return &remote, nil
}

// Update 执行更新
func (u *Updater) Update(remote *RemoteVersion) error {
	// 获取当前平台
	platform := u.getPlatform()

	// 查找下载 URL
	downloadURL, ok := remote.DownloadURL[platform]
	if !ok {
		// 如果没有指定平台的 URL，尝试使用默认 URL
		downloadURL = u.buildDownloadURL(platform)
	}

	// 下载新版本
	fmt.Printf("正在下载 %s ...\n", remote.String())
	tempFile, err := u.downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(tempFile)

	// 替换当前二进制文件
	if err := u.replaceBinary(tempFile); err != nil {
		return fmt.Errorf("替换二进制文件失败: %w", err)
	}

	fmt.Printf("✓ 更新成功: %s → %s\n", Current.String(), remote.String())
	return nil
}

// getPlatform 获取当前平台标识
func (u *Updater) getPlatform() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	return fmt.Sprintf("%s-%s", goos, goarch)
}

// buildDownloadURL 构建下载 URL
func (u *Updater) buildDownloadURL(platform string) string {
	// 平台到文件名的映射
	fileNameMap := map[string]string{
		"windows-amd64": "jiasinecli-windows.exe",
		"windows-arm64": "jiasinecli-windows-arm64.exe",
		"linux-amd64":   "jiasinecli-linux",
		"linux-arm64":   "jiasinecli-linux-arm64",
		"linux-arm":     "jiasinecli-raspi",
		"darwin-arm64":  "jiasinecli-macos-arm",
		"darwin-amd64":  "jiasinecli-macos-intel",
	}

	fileName, ok := fileNameMap[platform]
	if !ok {
		// 降级：使用默认命名规则
		if runtime.GOOS == "windows" {
			fileName = "jiasinecli-" + platform + ".exe"
		} else {
			fileName = "jiasinecli-" + platform
		}
	}

	return binaryDownloadPrefix + fileName
}

// downloadBinary 下载二进制文件到临时目录
func (u *Updater) downloadBinary(url string) (string, error) {
	resp, err := u.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，HTTP %d", resp.StatusCode)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "jiasinecli-update-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 下载到临时文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// replaceBinary 替换当前二进制文件
func (u *Updater) replaceBinary(newBinary string) error {
	// 获取当前二进制文件路径
	currentBinary, err := os.Executable()
	if err != nil {
		return err
	}

	// 解析符号链接
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return err
	}

	// 备份当前二进制
	backupPath := currentBinary + ".old"
	if err := os.Rename(currentBinary, backupPath); err != nil {
		return fmt.Errorf("备份失败: %w", err)
	}

	// 复制新二进制
	if err := u.copyFile(newBinary, currentBinary); err != nil {
		// 恢复备份
		os.Rename(backupPath, currentBinary)
		return fmt.Errorf("复制新版本失败: %w", err)
	}

	// 设置执行权限（Unix-like 系统）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(currentBinary, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %w", err)
		}
	}

	// 删除备份
	os.Remove(backupPath)
	return nil
}

// copyFile 复制文件
func (u *Updater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
