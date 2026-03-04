/*
 * Jiasine CLI - Objective-C 测试动态库
 * 使用纯 C 兼容接口导出，跨平台兼容 (无需 ObjC Runtime)
 *
 * 编译方式:
 *   macOS:   clang -shared -fPIC -framework Foundation -o libjiasine_objc_test.dylib JiasineObjcTest.m
 *   Linux:   gcc -shared -fPIC -lobjc -o libjiasine_objc_test.so JiasineObjcTest.m $(gnustep-config --objc-flags --base-libs)
 *   Windows: gcc -shared -o jiasine_objc_test.dll JiasineObjcTest.m -lobjc (需要 GNUstep 或 MinGW ObjC 支持)
 *
 * 注意: 此文件提供两种编译模式:
 *   1. 完整 ObjC 模式: macOS/Linux with ObjC runtime
 *   2. 纯 C 兼容模式: 当 ObjC 不可用时回退为纯 C 实现 (OBJC_FALLBACK_C)
 *
 * 默认自动检测: 如果编译器支持 ObjC 则用 ObjC, 否则用纯 C
 */

#if defined(__OBJC__) && !defined(OBJC_FALLBACK_C)
/* ═══════════════ Objective-C 实现 ═══════════════ */

#import <Foundation/Foundation.h>
#include <string.h>

#ifdef _WIN32
    #define EXPORT __declspec(dllexport)
#else
    #define EXPORT __attribute__((visibility("default")))
#endif

static char result_buf[2048];

@interface JiasineHelper : NSObject
+ (NSString *)addWithInput:(NSString *)input;
+ (NSString *)reverseString:(NSString *)input;
+ (NSString *)getVersion;
+ (NSString *)health;
@end

@implementation JiasineHelper
+ (NSString *)addWithInput:(NSString *)input {
    NSArray *parts = [input componentsSeparatedByString:@" "];
    if (parts.count < 2) return @"{\"error\": \"需要2个参数\"}";
    int a = [parts[0] intValue];
    int b = [parts[1] intValue];
    return [NSString stringWithFormat:@"{\"result\": %d, \"lang\": \"Objective-C\"}", a + b];
}

+ (NSString *)reverseString:(NSString *)input {
    NSMutableString *reversed = [NSMutableString string];
    for (NSInteger i = input.length - 1; i >= 0; i--) {
        [reversed appendFormat:@"%C", [input characterAtIndex:i]];
    }
    return [NSString stringWithFormat:@"{\"input\": \"%@\", \"reversed\": \"%@\", \"lang\": \"Objective-C\"}", input, reversed];
}

+ (NSString *)getVersion {
    return @"{\"name\": \"jiasine_objc_test\", \"version\": \"1.0.0\", \"lang\": \"Objective-C\"}";
}

+ (NSString *)health {
    return @"{\"status\": \"ok\", \"lang\": \"Objective-C\"}";
}
@end

EXPORT const char* add(const char* input) {
    @autoreleasepool {
        NSString *inputStr = input ? [NSString stringWithUTF8String:input] : @"";
        NSString *result = [JiasineHelper addWithInput:inputStr];
        strncpy(result_buf, [result UTF8String], sizeof(result_buf) - 1);
        return result_buf;
    }
}

EXPORT const char* get_version(void) {
    @autoreleasepool {
        NSString *result = [JiasineHelper getVersion];
        strncpy(result_buf, [result UTF8String], sizeof(result_buf) - 1);
        return result_buf;
    }
}

EXPORT const char* reverse_string(const char* input) {
    @autoreleasepool {
        NSString *inputStr = input ? [NSString stringWithUTF8String:input] : @"";
        NSString *result = [JiasineHelper reverseString:inputStr];
        strncpy(result_buf, [result UTF8String], sizeof(result_buf) - 1);
        return result_buf;
    }
}

EXPORT const char* health(void) {
    return "{\"status\": \"ok\", \"lang\": \"Objective-C\"}";
}

#else
/* ═══════════════ 纯 C 兼容回退实现 ═══════════════ */
/* 当 ObjC 运行时不可用时使用此实现 (Windows/MinGW 等) */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
    #define EXPORT __declspec(dllexport)
#else
    #define EXPORT __attribute__((visibility("default")))
#endif

static char result_buf[2048];

EXPORT const char* add(const char* input) {
    int a = 0, b = 0;
    if (input != NULL) {
        sscanf(input, "%d %d", &a, &b);
    }
    snprintf(result_buf, sizeof(result_buf),
        "{\"result\": %d, \"lang\": \"Objective-C\"}", a + b);
    return result_buf;
}

EXPORT const char* get_version(void) {
    snprintf(result_buf, sizeof(result_buf),
        "{\"name\": \"jiasine_objc_test\", \"version\": \"1.0.0\", \"lang\": \"Objective-C\"}");
    return result_buf;
}

EXPORT const char* reverse_string(const char* input) {
    if (input == NULL) {
        snprintf(result_buf, sizeof(result_buf), "{\"error\": \"null input\"}");
        return result_buf;
    }

    int len = (int)strlen(input);
    if (len >= (int)sizeof(result_buf) - 100) {
        len = (int)sizeof(result_buf) - 100;
    }

    char reversed[1024];
    for (int i = 0; i < len; i++) {
        reversed[i] = input[len - 1 - i];
    }
    reversed[len] = '\0';

    snprintf(result_buf, sizeof(result_buf),
        "{\"input\": \"%s\", \"reversed\": \"%s\", \"lang\": \"Objective-C\"}", input, reversed);
    return result_buf;
}

EXPORT const char* health(void) {
    return "{\"status\": \"ok\", \"lang\": \"Objective-C\"}";
}

#endif
