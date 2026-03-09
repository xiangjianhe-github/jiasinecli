package version

import (
	"strings"
	"testing"
)

func TestBuildDownloadURL(t *testing.T) {
	updater := NewUpdater()

	tests := []struct {
		platform    string
		expectedURL string
	}{
		{"windows-amd64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-windows.exe"},
		{"windows-arm64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-windows-arm64.exe"},
		{"linux-amd64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-linux"},
		{"linux-arm64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-linux-arm64"},
		{"linux-arm", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-raspi"},
		{"darwin-arm64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-macos-arm"},
		{"darwin-amd64", "https://drive.cvte.com/p/DZjGLnIQosoIGOLAZCAA/jiasinecli-macos-intel"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			url := updater.buildDownloadURL(tt.platform)
			if url != tt.expectedURL {
				t.Errorf("buildDownloadURL(%s) = %s, want %s", tt.platform, url, tt.expectedURL)
			}
		})
	}
}

func TestGetPlatform(t *testing.T) {
	updater := NewUpdater()
	platform := updater.getPlatform()

	// 验证格式: OS-ARCH
	if !strings.Contains(platform, "-") {
		t.Errorf("getPlatform() = %s, expected format 'OS-ARCH'", platform)
	}
}
