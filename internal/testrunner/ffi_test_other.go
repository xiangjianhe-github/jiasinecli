// 非 Windows 平台 DLL 测试 stub

//go:build !windows

package testrunner

func testCallDLLPlatform(r *Runner, dllPath string) {
	r.addResult("FFI", "DLL调用", false, "", "直接 DLL 测试仅支持 Windows，其他平台请使用 CGO")
}

func testCallDLLWithLangPlatform(r *Runner, dllPath string, lang string) {
	r.addResult(lang, "DLL调用", false, "", "直接 DLL 测试仅支持 Windows，其他平台请使用 CGO")
}
