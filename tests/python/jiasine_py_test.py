"""
Jiasine CLI - Python 测试服务
一个简单的 HTTP 服务，用于验证 CLI 的 Service 层调用 Python 能力

用法:
    python jiasine_py_test.py              # 启动 HTTP 服务 (端口 9901)
    python jiasine_py_test.py --port 8080  # 指定端口

测试:
    curl http://localhost:9901/health
    curl -X POST http://localhost:9901/add -d '{"params":["3","5"]}'
    curl -X POST http://localhost:9901/reverse -d '{"params":["hello"]}'
    curl -X POST http://localhost:9901/fibonacci -d '{"params":["10"]}'
"""

import json
import sys
import argparse
from http.server import HTTPServer, BaseHTTPRequestHandler


class JiasineHandler(BaseHTTPRequestHandler):
    """Jiasine 测试服务 Handler"""

    def do_GET(self):
        """处理 GET 请求"""
        if self.path == "/health":
            self._respond(200, {"status": "ok", "lang": "Python", "version": sys.version})
        elif self.path == "/version":
            self._respond(200, {
                "name": "jiasine_py_test",
                "version": "1.0.0",
                "lang": "Python",
                "python_version": sys.version
            })
        else:
            self._respond(404, {"error": f"未知路径: {self.path}"})

    def do_POST(self):
        """处理 POST 请求"""
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length)

        try:
            data = json.loads(body) if body else {}
        except json.JSONDecodeError:
            self._respond(400, {"error": "无效的 JSON"})
            return

        params = data.get("params", [])
        path = self.path.strip("/")

        handlers = {
            "add": self._handle_add,
            "reverse": self._handle_reverse,
            "fibonacci": self._handle_fibonacci,
            "upper": self._handle_upper,
        }

        handler = handlers.get(path)
        if handler:
            try:
                result = handler(params)
                self._respond(200, result)
            except Exception as e:
                self._respond(500, {"error": str(e), "lang": "Python"})
        else:
            self._respond(404, {"error": f"未知方法: {path}", "available": list(handlers.keys())})

    def _handle_add(self, params):
        """加法: params = ["3", "5"]"""
        if len(params) < 2:
            raise ValueError("需要至少2个参数")
        a, b = int(params[0]), int(params[1])
        return {"result": a + b, "lang": "Python"}

    def _handle_reverse(self, params):
        """字符串反转"""
        if not params:
            raise ValueError("需要至少1个参数")
        text = params[0]
        return {"input": text, "reversed": text[::-1], "lang": "Python"}

    def _handle_fibonacci(self, params):
        """斐波那契数列"""
        if not params:
            raise ValueError("需要1个参数 (数列长度)")
        n = int(params[0])
        fib = []
        a, b = 0, 1
        for _ in range(n):
            fib.append(a)
            a, b = b, a + b
        return {"n": n, "fibonacci": fib, "lang": "Python"}

    def _handle_upper(self, params):
        """转大写"""
        if not params:
            raise ValueError("需要至少1个参数")
        text = " ".join(params)
        return {"input": text, "upper": text.upper(), "lang": "Python"}

    def _respond(self, status, data):
        """发送 JSON 响应"""
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.end_headers()
        self.wfile.write(json.dumps(data, ensure_ascii=False).encode("utf-8"))

    def log_message(self, format, *args):
        """自定义日志格式"""
        print(f"[Python Service] {self.client_address[0]} - {format % args}")


def main():
    parser = argparse.ArgumentParser(description="Jiasine Python 测试服务")
    parser.add_argument("--port", type=int, default=9901, help="监听端口 (默认 9901)")
    parser.add_argument("--host", type=str, default="127.0.0.1", help="监听地址 (默认 127.0.0.1)")
    args = parser.parse_args()

    server = HTTPServer((args.host, args.port), JiasineHandler)
    print(f"[Jiasine Python Test Service] 启动于 http://{args.host}:{args.port}")
    print(f"  GET  /health       - 健康检查")
    print(f"  GET  /version      - 版本信息")
    print(f"  POST /add          - 加法运算")
    print(f"  POST /reverse      - 字符串反转")
    print(f"  POST /fibonacci    - 斐波那契数列")
    print(f"  POST /upper        - 转大写")
    print(f"按 Ctrl+C 停止...")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n服务已停止")
        server.server_close()


if __name__ == "__main__":
    main()
