@echo off
REM Jiasine CLI - C 测试库编译脚本 (Windows)
REM 需要安装 GCC (MinGW) 或 MSVC

echo [编译 C 测试动态库]

REM 尝试 GCC
where gcc >nul 2>&1
if %ERRORLEVEL%==0 (
    echo 使用 GCC 编译...
    gcc -shared -o jiasine_c_test.dll jiasine_c_test.c -Wl,--out-implib,libjiasine_c_test.a
    if %ERRORLEVEL%==0 (
        echo 编译成功: jiasine_c_test.dll
    ) else (
        echo 编译失败!
    )
    goto :end
)

REM 尝试 MSVC cl
where cl >nul 2>&1
if %ERRORLEVEL%==0 (
    echo 使用 MSVC 编译...
    cl /LD jiasine_c_test.c /Fe:jiasine_c_test.dll
    if %ERRORLEVEL%==0 (
        echo 编译成功: jiasine_c_test.dll
    ) else (
        echo 编译失败!
    )
    goto :end
)

echo 错误: 未找到 GCC 或 MSVC 编译器
echo 请安装 MinGW-w64 或 Visual Studio Build Tools

:end
