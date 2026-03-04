/*
 * Jiasine CLI - C 语言测试动态库
 * 编译方式:
 *   Windows: cl /LD jiasine_c_test.c /Fe:jiasine_c_test.dll
 *            或 gcc -shared -o jiasine_c_test.dll jiasine_c_test.c
 *   Linux:   gcc -shared -fPIC -o libjiasine_c_test.so jiasine_c_test.c
 *   macOS:   gcc -shared -fPIC -o libjiasine_c_test.dylib jiasine_c_test.c
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
    #define EXPORT __declspec(dllexport)
#else
    #define EXPORT __attribute__((visibility("default")))
#endif

// 静态缓冲区用于返回字符串
static char result_buf[1024];

// 加法运算 - 接收 "a b" 格式字符串，返回结果字符串
EXPORT const char* add(const char* input) {
    int a = 0, b = 0;
    if (input != NULL) {
        sscanf(input, "%d %d", &a, &b);
    }
    snprintf(result_buf, sizeof(result_buf), "{\"result\": %d, \"lang\": \"C\"}", a + b);
    return result_buf;
}

// 获取版本信息
EXPORT const char* get_version(void) {
    snprintf(result_buf, sizeof(result_buf),
        "{\"name\": \"jiasine_c_test\", \"version\": \"1.0.0\", \"lang\": \"C\"}");
    return result_buf;
}

// 字符串反转
EXPORT const char* reverse_string(const char* input) {
    if (input == NULL) {
        snprintf(result_buf, sizeof(result_buf), "{\"error\": \"null input\"}");
        return result_buf;
    }

    int len = (int)strlen(input);
    if (len >= (int)sizeof(result_buf) - 50) {
        len = (int)sizeof(result_buf) - 50;
    }

    char reversed[512];
    for (int i = 0; i < len; i++) {
        reversed[i] = input[len - 1 - i];
    }
    reversed[len] = '\0';

    snprintf(result_buf, sizeof(result_buf),
        "{\"input\": \"%s\", \"reversed\": \"%s\", \"lang\": \"C\"}", input, reversed);
    return result_buf;
}

// 健康检查
EXPORT const char* health(void) {
    return "{\"status\": \"ok\", \"lang\": \"C\"}";
}
