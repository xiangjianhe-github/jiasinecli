// Package winres 生成 Windows .exe 的资源文件（图标、版本信息、清单）
//
// 运行方式:
//   go run internal/winres/generate.go
//
// 生成产物:
//   rsrc_windows_amd64.syso  (嵌入到 go build 中)
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	fmt.Println("生成 Windows 资源文件...")

	// 生成多尺寸 ICO (16x16, 32x32, 48x48, 256x256)
	sizes := []int{16, 32, 48, 256}
	if err := generateICO("assets/app.ico", sizes); err != nil {
		fmt.Printf("生成 ICO 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ assets/app.ico")

	// 生成 .rc 资源脚本
	if err := generateRC("assets/app.rc"); err != nil {
		fmt.Printf("生成 RC 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ assets/app.rc")

	// 生成 Windows 应用清单 (启用 DPI 感知、UAC 信息)
	if err := generateManifest("assets/app.manifest"); err != nil {
		fmt.Printf("生成 manifest 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ assets/app.manifest")

	fmt.Printf("\n下一步: 使用 windres 编译资源\n")
	fmt.Printf("  windres -o rsrc_windows_%s.syso assets/app.rc\n", runtime.GOARCH)
}

// generateICO 创建 ICO 文件，包含多个尺寸的 PNG 图标
// 图标设计：仿 icon.json Lottie 中的六边形 + J 字母
func generateICO(path string, sizes []int) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	var entries []icoEntry
	var pngDataList [][]byte

	offset := uint32(6 + 16*len(sizes)) // ICO header + entry headers

	for _, size := range sizes {
		img := drawIcon(size)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		data := buf.Bytes()
		pngDataList = append(pngDataList, data)

		w, h := byte(size), byte(size)
		if size >= 256 {
			w, h = 0, 0 // 256+ 用 0 表示
		}

		entries = append(entries, icoEntry{
			Width:      w,
			Height:     h,
			ColorCount: 0,
			Reserved:   0,
			Planes:     1,
			BitCount:   32,
			Size:       uint32(len(data)),
			Offset:     offset,
		})
		offset += uint32(len(data))
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// ICO header
	binary.Write(f, binary.LittleEndian, icoHeader{
		Reserved: 0,
		Type:     1,
		Count:    uint16(len(sizes)),
	})
	// Entry headers
	for _, e := range entries {
		binary.Write(f, binary.LittleEndian, e)
	}
	// PNG data
	for _, data := range pngDataList {
		f.Write(data)
	}

	return nil
}

type icoHeader struct {
	Reserved uint16
	Type     uint16
	Count    uint16
}

type icoEntry struct {
	Width      byte
	Height     byte
	ColorCount byte
	Reserved   byte
	Planes     uint16
	BitCount   uint16
	Size       uint32
	Offset     uint32
}

// drawIcon 绘制图标 — 六边形背景 + "J" 字母
// 配色来自 icon.json 的 Lottie 动画
func drawIcon(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2
	radius := float64(size) * 0.42

	// 配色
	bgColor := color.RGBA{93, 175, 240, 255}     // #5DAFF0 蓝
	fgColor := color.RGBA{94, 248, 219, 255}      // #5EF8DB 青
	accentColor := color.RGBA{49, 201, 227, 255}   // #31C9E3 青蓝

	// 绘制圆形背景
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := dx*dx + dy*dy
			if dist <= radius*radius {
				// 渐变效果
				t := dist / (radius * radius)
				r := lerp(float64(bgColor.R), float64(accentColor.R), t)
				g := lerp(float64(bgColor.G), float64(accentColor.G), t)
				b := lerp(float64(bgColor.B), float64(accentColor.B), t)
				img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
			}
		}
	}

	// 绘制 "J" 字母
	drawJ(img, size, fgColor)

	return img
}

func drawJ(img *image.RGBA, size int, col color.RGBA) {
	s := float64(size)
	// J 的各段：顶部横线、右竖线、底部弧线
	thick := s * 0.12

	// 顶横线 (J 顶部)
	for y := int(s * 0.22); y < int(s*0.22+thick); y++ {
		for x := int(s * 0.30); x < int(s*0.70); x++ {
			img.Set(x, y, col)
		}
	}
	// 右竖线
	for y := int(s * 0.22); y < int(s*0.65); y++ {
		for x := int(s*0.55 - thick/2); x < int(s*0.55+thick/2); x++ {
			img.Set(x, y, col)
		}
	}
	// 底部弧线 (简化为方形圆角)
	cx, cy := s*0.42, s*0.65
	r := s * 0.13
	for y := int(cy - r); y < int(cy+r+thick); y++ {
		for x := int(cx - r); x < int(cx+r); x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := dx*dx + dy*dy
			if dist >= (r-thick)*(r-thick) && dist <= (r+thick/2)*(r+thick/2) {
				if float64(y) >= cy { // 只画下半弧
					img.Set(x, y, col)
				}
			}
		}
	}
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func generateRC(path string) error {
	rc := `// Windows 资源脚本 - 自动生成
// 编译: windres -o rsrc_windows_amd64.syso assets/app.rc

#include <winver.h>

// 应用图标 (ID=1 是主图标)
1 ICON "app.ico"

// 应用清单
1 24 "app.manifest"

// 版本信息
VS_VERSION_INFO VERSIONINFO
FILEVERSION     0,1,0,0
PRODUCTVERSION  0,1,0,0
FILEFLAGSMASK   VS_FFI_FILEFLAGSMASK
FILEFLAGS       0
FILEOS          VOS_NT_WINDOWS32
FILETYPE        VFT_APP
FILESUBTYPE     0
BEGIN
    BLOCK "StringFileInfo"
    BEGIN
        BLOCK "040904B0"
        BEGIN
            VALUE "CompanyName",      "Jiasine"
            VALUE "FileDescription",  "Jiasine CLI - Cross-platform multi-language support system"
            VALUE "FileVersion",      "0.1.0-alpha.1"
            VALUE "InternalName",     "jiasinecli"
            VALUE "LegalCopyright",   "Copyright (c) 2026 Jiasine"
            VALUE "OriginalFilename", "jiasinecli.exe"
            VALUE "ProductName",      "Jiasine CLI"
            VALUE "ProductVersion",   "0.1.0-alpha.1"
        END
    END
    BLOCK "VarFileInfo"
    BEGIN
        VALUE "Translation", 0x0409, 0x04B0
    END
END
`
	return os.WriteFile(path, []byte(rc), 0644)
}

func generateManifest(path string) error {
	manifest := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity
    type="win32"
    name="Jiasine.CLI"
    version="0.1.0.0"
    processorArchitecture="*"/>
  <description>Jiasine CLI - Cross-platform multi-language support system</description>

  <!-- 请求管理员权限时不需要提升 -->
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>

  <!-- DPI 感知 -->
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
    </windowsSettings>
  </application>

  <!-- 兼容性 -->
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <!-- Windows 10/11 -->
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
    </application>
  </compatibility>
</assembly>
`
	return os.WriteFile(path, []byte(manifest), 0644)
}
