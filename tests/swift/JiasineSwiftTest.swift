/**
 * Jiasine CLI - Swift 测试服务
 * 使用 Swift Foundation 的简易 HTTP 服务器
 *
 * 编译: swiftc -o jiasine_swift_test JiasineSwiftTest.swift
 * 运行: ./jiasine_swift_test              # 端口 9906
 *       ./jiasine_swift_test --port 8080  # 自定义端口
 *
 * 测试:
 *     curl http://localhost:9906/health
 *     curl -X POST http://localhost:9906/add -d '{"params":["3","5"]}'
 *
 * 注意: 此实现使用 BSD Socket + Foundation, 无需第三方依赖
 */

import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif

// MARK: - 业务逻辑

func handleAdd(_ params: [String]) -> [String: Any] {
    guard params.count >= 2,
          let a = Int(params[0]),
          let b = Int(params[1]) else {
        return ["error": "需要至少2个参数"]
    }
    return ["result": a + b, "lang": "Swift"]
}

func handleReverse(_ params: [String]) -> [String: Any] {
    guard let text = params.first else {
        return ["error": "需要至少1个参数"]
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

func handleFactorial(_ params: [String]) -> [String: Any] {
    guard let first = params.first, let n = Int(first) else {
        return ["error": "需要1个参数"]
    }
    var result = 1
    for i in 2...max(2, n) { result *= i }
    return ["n": n, "factorial": result, "lang": "Swift"]
}

// MARK: - 简易 HTTP 服务器 (基于 Socket)

class SimpleHTTPServer {
    let port: UInt16
    let host: String
    private var serverSocket: Int32 = -1

    init(host: String = "127.0.0.1", port: UInt16 = 9906) {
        self.host = host
        self.port = port
    }

    func start() {
        #if os(Linux) || os(Windows)
        serverSocket = socket(AF_INET, Int32(SOCK_STREAM.rawValue), 0)
        #else
        serverSocket = socket(AF_INET, SOCK_STREAM, 0)
        #endif

        guard serverSocket >= 0 else {
            print("[Swift Service] 创建 socket 失败")
            return
        }

        var reuse: Int32 = 1
        setsockopt(serverSocket, SOL_SOCKET, SO_REUSEADDR, &reuse, socklen_t(MemoryLayout<Int32>.size))

        var addr = sockaddr_in()
        addr.sin_family = sa_family_t(AF_INET)
        addr.sin_port = port.bigEndian
        addr.sin_addr.s_addr = inet_addr(host)

        let bindResult = withUnsafePointer(to: &addr) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                bind(serverSocket, $0, socklen_t(MemoryLayout<sockaddr_in>.size))
            }
        }

        guard bindResult == 0 else {
            print("[Swift Service] 绑定端口 \(port) 失败")
            return
        }

        listen(serverSocket, 10)
        print("[Swift Service] 启动于 http://\(host):\(port)")

        while true {
            var clientAddr = sockaddr_in()
            var clientAddrLen = socklen_t(MemoryLayout<sockaddr_in>.size)
            let clientSocket = withUnsafeMutablePointer(to: &clientAddr) {
                $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                    accept(serverSocket, $0, &clientAddrLen)
                }
            }

            guard clientSocket >= 0 else { continue }

            // 读取请求
            var buffer = [UInt8](repeating: 0, count: 65536)
            let bytesRead = recv(clientSocket, &buffer, buffer.count, 0)
            guard bytesRead > 0 else {
                close(clientSocket)
                continue
            }

            let requestStr = String(bytes: buffer[0..<bytesRead], encoding: .utf8) ?? ""
            let response = handleRequest(requestStr)

            // 发送响应
            let httpResponse = "HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\nContent-Length: \(response.utf8.count)\r\nConnection: close\r\n\r\n\(response)"
            _ = httpResponse.withCString { send(clientSocket, $0, httpResponse.utf8.count, 0) }
            close(clientSocket)
        }
    }

    func handleRequest(_ request: String) -> String {
        let lines = request.components(separatedBy: "\r\n")
        guard let firstLine = lines.first else {
            return toJSON(["error": "空请求"])
        }

        let parts = firstLine.components(separatedBy: " ")
        guard parts.count >= 2 else {
            return toJSON(["error": "无效请求"])
        }

        let method = parts[0]
        let path = parts[1].trimmingCharacters(in: CharacterSet(charactersIn: "/"))

        // GET 请求
        if method == "GET" {
            switch path {
            case "health":
                return toJSON(["status": "ok", "lang": "Swift"])
            case "version":
                return toJSON(["name": "jiasine_swift_test", "version": "1.0.0", "lang": "Swift"])
            default:
                return toJSON(["error": "未知路径: /\(path)"])
            }
        }

        // POST 请求 - 解析 body
        if method == "POST" {
            // 找到空行后的 body
            let bodyStart = request.range(of: "\r\n\r\n")
            let body = bodyStart.map { String(request[$0.upperBound...]) } ?? ""
            let params = parseParams(body)

            let handlers: [String: ([String]) -> [String: Any]] = [
                "add": handleAdd,
                "reverse": handleReverse,
                "fibonacci": handleFibonacci,
                "factorial": handleFactorial
            ]

            guard let handler = handlers[path] else {
                return toJSON(["error": "未知方法: \(path)", "available": Array(handlers.keys).description])
            }

            let result = handler(params)
            return toJSON(result)
        }

        return toJSON(["error": "Method not allowed"])
    }

    func parseParams(_ body: String) -> [String] {
        // 简易解析 {"params": ["a", "b"]}
        guard let start = body.range(of: "["),
              let end = body.range(of: "]") else { return [] }

        let arrStr = String(body[start.upperBound..<end.lowerBound])
        return arrStr.components(separatedBy: ",")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .map { $0.trimmingCharacters(in: CharacterSet(charactersIn: "\"")) }
            .filter { !$0.isEmpty }
    }

    func toJSON(_ dict: [String: Any]) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: dict, options: []),
              let str = String(data: data, encoding: .utf8) else {
            return "{\"error\": \"JSON 序列化失败\"}"
        }
        return str
    }
}

// MARK: - 入口

var port: UInt16 = 9906
var host = "127.0.0.1"
let arguments = CommandLine.arguments

for i in 1..<arguments.count {
    if arguments[i] == "--port" && i + 1 < arguments.count {
        port = UInt16(arguments[i + 1]) ?? 9906
    } else if arguments[i] == "--host" && i + 1 < arguments.count {
        host = arguments[i + 1]
    }
}

let server = SimpleHTTPServer(host: host, port: port)
server.start()
