/**
 * Jiasine CLI - Swift 进程调用测试
 * 通过子进程方式调用 (service type: process)
 *
 * 编译: swiftc -o jiasine_swift_process JiasineSwiftProcess.swift
 * 用法:
 *     ./jiasine_swift_process add 3 5
 *     ./jiasine_swift_process reverse hello
 *     ./jiasine_swift_process fibonacci 10
 *
 * macOS 也可直接脚本执行:
 *     swift JiasineSwiftProcess.swift add 3 5
 */

import Foundation

func handleAdd(_ params: [String]) -> [String: Any] {
    guard params.count >= 2,
          let a = Int(params[0]),
          let b = Int(params[1]) else {
        return ["error": "需要2个参数"]
    }
    return ["result": a + b, "lang": "Swift"]
}

func handleReverse(_ params: [String]) -> [String: Any] {
    guard let text = params.first else {
        return ["error": "需要1个参数"]
    }
    return ["input": text, "reversed": String(text.reversed()), "lang": "Swift"]
}

func handleFibonacci(_ params: [String]) -> [String: Any] {
    guard let first = params.first, let n = Int(first) else {
        return ["error": "需要1个参数"]
    }
    var fib: [Int] = []
    var a = 0, b = 1
    for _ in 0..<n {
        fib.append(a)
        let temp = a
        a = b
        b = temp + b
    }
    return ["n": n, "fibonacci": fib, "lang": "Swift"]
}

func toJSON(_ dict: [String: Any]) -> String {
    guard let data = try? JSONSerialization.data(withJSONObject: dict, options: []),
          let str = String(data: data, encoding: .utf8) else {
        return "{\"error\": \"JSON 序列化失败\"}"
    }
    return str
}

// MARK: - 入口

let arguments = CommandLine.arguments

guard arguments.count >= 2 else {
    print("{\"error\": \"用法: jiasine_swift_process <method> [args...]\"}")
    exit(1)
}

let method = arguments[1]
let params = Array(arguments[2...])

let handlers: [String: ([String]) -> [String: Any]] = [
    "add": handleAdd,
    "reverse": handleReverse,
    "fibonacci": handleFibonacci
]

guard let handler = handlers[method] else {
    let available = Array(handlers.keys)
    print("{\"error\": \"未知方法: \(method)\", \"available\": \(available)}")
    exit(1)
}

let result = handler(params)
print(toJSON(result))
