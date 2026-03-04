#!/bin/bash
# Jiasine CLI - C 测试库编译脚本 (Linux/macOS)

echo "[编译 C 测试动态库]"

OS=$(uname -s)
case "$OS" in
    Linux)
        echo "编译 Linux .so ..."
        gcc -shared -fPIC -o libjiasine_c_test.so jiasine_c_test.c
        echo "完成: libjiasine_c_test.so"
        ;;
    Darwin)
        echo "编译 macOS .dylib ..."
        gcc -shared -fPIC -o libjiasine_c_test.dylib jiasine_c_test.c
        echo "完成: libjiasine_c_test.dylib"
        ;;
    *)
        echo "不支持的平台: $OS"
        exit 1
        ;;
esac
