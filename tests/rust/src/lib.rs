/*
 * Jiasine CLI - Rust 测试动态库
 *
 * 编译方式:
 *   cargo build --release
 *
 * 产物:
 *   Windows: target/release/jiasine_rust_test.dll
 *   Linux:   target/release/libjiasine_rust_test.so
 *   macOS:   target/release/libjiasine_rust_test.dylib
 */

use std::ffi::{CStr, CString};
use std::os::raw::c_char;

/// 静态结果缓冲 (简化示例用, 生产环境应使用线程安全方案)
static mut RESULT_BUF: [u8; 1024] = [0u8; 1024];

/// 将 Rust String 写入静态缓冲并返回指针
unsafe fn set_result(s: &str) -> *const c_char {
    let bytes = s.as_bytes();
    let len = bytes.len().min(RESULT_BUF.len() - 1);
    RESULT_BUF[..len].copy_from_slice(&bytes[..len]);
    RESULT_BUF[len] = 0; // null terminator
    RESULT_BUF.as_ptr() as *const c_char
}

/// 从 C 字符串指针读取 Rust String
unsafe fn read_input(input: *const c_char) -> String {
    if input.is_null() {
        return String::new();
    }
    CStr::from_ptr(input).to_string_lossy().into_owned()
}

/// 加法运算 - 输入格式 "a b"
#[no_mangle]
pub unsafe extern "C" fn add(input: *const c_char) -> *const c_char {
    let s = read_input(input);
    let parts: Vec<&str> = s.split_whitespace().collect();

    let (a, b) = if parts.len() >= 2 {
        (
            parts[0].parse::<i64>().unwrap_or(0),
            parts[1].parse::<i64>().unwrap_or(0),
        )
    } else {
        (0, 0)
    };

    let result = format!(r#"{{"result": {}, "lang": "Rust"}}"#, a + b);
    set_result(&result)
}

/// 获取版本信息
#[no_mangle]
pub unsafe extern "C" fn get_version() -> *const c_char {
    set_result(r#"{"name": "jiasine_rust_test", "version": "1.0.0", "lang": "Rust"}"#)
}

/// 字符串反转
#[no_mangle]
pub unsafe extern "C" fn reverse_string(input: *const c_char) -> *const c_char {
    let s = read_input(input);
    let reversed: String = s.chars().rev().collect();
    let result = format!(
        r#"{{"input": "{}", "reversed": "{}", "lang": "Rust"}}"#,
        s, reversed
    );
    set_result(&result)
}

/// SHA256 哈希 (简化版 — 实际可用 sha2 crate)
#[no_mangle]
pub unsafe extern "C" fn hash(input: *const c_char) -> *const c_char {
    let s = read_input(input);
    // 简单哈希示例 (DJB2 算法)
    let mut h: u64 = 5381;
    for b in s.bytes() {
        h = h.wrapping_mul(33).wrapping_add(b as u64);
    }
    let result = format!(
        r#"{{"input": "{}", "hash": "{:016x}", "algorithm": "djb2", "lang": "Rust"}}"#,
        s, h
    );
    set_result(&result)
}

/// 健康检查
#[no_mangle]
pub unsafe extern "C" fn health() -> *const c_char {
    set_result(r#"{"status": "ok", "lang": "Rust"}"#)
}
