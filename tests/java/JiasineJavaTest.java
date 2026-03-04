/**
 * Jiasine CLI - Java 测试服务
 * 使用 JDK 内置 HttpServer，无外部依赖
 *
 * 编译: javac JiasineJavaTest.java
 * 运行: java JiasineJavaTest              # 端口 9905
 *       java JiasineJavaTest --port 8080  # 自定义端口
 *
 * 测试:
 *     curl http://localhost:9905/health
 *     curl -X POST http://localhost:9905/add -d '{"params":["3","5"]}'
 */

import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpExchange;
import java.io.*;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.stream.Collectors;

public class JiasineJavaTest {

    public static void main(String[] args) throws IOException {
        int port = 9905;
        String host = "127.0.0.1";

        for (int i = 0; i < args.length; i++) {
            if ("--port".equals(args[i]) && i + 1 < args.length) {
                port = Integer.parseInt(args[++i]);
            } else if ("--host".equals(args[i]) && i + 1 < args.length) {
                host = args[++i];
            }
        }

        HttpServer server = HttpServer.create(new InetSocketAddress(host, port), 0);
        server.createContext("/health", new HealthHandler());
        server.createContext("/version", new VersionHandler());
        server.createContext("/add", new AddHandler());
        server.createContext("/reverse", new ReverseHandler());
        server.createContext("/fibonacci", new FibonacciHandler());
        server.createContext("/factorial", new FactorialHandler());
        server.setExecutor(null);
        server.start();

        System.out.printf("[Java Service] 启动于 http://%s:%d%n", host, port);
    }

    // ═══════ 工具方法 ═══════

    static void respond(HttpExchange exchange, int code, String json) throws IOException {
        byte[] bytes = json.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json; charset=utf-8");
        exchange.sendResponseHeaders(code, bytes.length);
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(bytes);
        }
    }

    static String readBody(HttpExchange exchange) throws IOException {
        try (BufferedReader reader = new BufferedReader(
                new InputStreamReader(exchange.getRequestBody(), StandardCharsets.UTF_8))) {
            return reader.lines().collect(Collectors.joining("\n"));
        }
    }

    static List<String> parseParams(String body) {
        // 简单 JSON 解析: {"params": ["a","b"]}
        List<String> params = new ArrayList<>();
        if (body == null || body.isEmpty()) return params;

        int idx = body.indexOf("[");
        int end = body.indexOf("]");
        if (idx < 0 || end < 0) return params;

        String arr = body.substring(idx + 1, end).trim();
        if (arr.isEmpty()) return params;

        for (String item : arr.split(",")) {
            String s = item.trim();
            if (s.startsWith("\"") && s.endsWith("\"")) {
                s = s.substring(1, s.length() - 1);
            }
            params.add(s);
        }
        return params;
    }

    static String jsonObj(String... kvPairs) {
        StringBuilder sb = new StringBuilder("{");
        for (int i = 0; i < kvPairs.length; i += 2) {
            if (i > 0) sb.append(", ");
            String key = kvPairs[i];
            String val = kvPairs[i + 1];
            sb.append("\"").append(key).append("\": ");
            // 数字或数组不加引号
            if (val.matches("-?\\d+") || val.startsWith("[") || val.startsWith("{")
                    || "true".equals(val) || "false".equals(val) || "null".equals(val)) {
                sb.append(val);
            } else {
                sb.append("\"").append(val).append("\"");
            }
        }
        sb.append("}");
        return sb.toString();
    }

    // ═══════ Handler 实现 ═══════

    static class HealthHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            respond(exchange, 200, jsonObj("status", "ok", "lang", "Java",
                    "version", System.getProperty("java.version")));
        }
    }

    static class VersionHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            respond(exchange, 200, jsonObj(
                    "name", "jiasine_java_test",
                    "version", "1.0.0",
                    "lang", "Java",
                    "java_version", System.getProperty("java.version")));
        }
    }

    static class AddHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            try {
                String body = readBody(exchange);
                List<String> params = parseParams(body);
                if (params.size() < 2) {
                    respond(exchange, 400, jsonObj("error", "需要至少2个参数"));
                    return;
                }
                int a = Integer.parseInt(params.get(0));
                int b = Integer.parseInt(params.get(1));
                respond(exchange, 200, jsonObj("result", String.valueOf(a + b), "lang", "Java"));
            } catch (Exception e) {
                respond(exchange, 500, jsonObj("error", e.getMessage(), "lang", "Java"));
            }
        }
    }

    static class ReverseHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            try {
                String body = readBody(exchange);
                List<String> params = parseParams(body);
                if (params.isEmpty()) {
                    respond(exchange, 400, jsonObj("error", "需要至少1个参数"));
                    return;
                }
                String text = params.get(0);
                String reversed = new StringBuilder(text).reverse().toString();
                respond(exchange, 200, jsonObj("input", text, "reversed", reversed, "lang", "Java"));
            } catch (Exception e) {
                respond(exchange, 500, jsonObj("error", e.getMessage(), "lang", "Java"));
            }
        }
    }

    static class FibonacciHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            try {
                String body = readBody(exchange);
                List<String> params = parseParams(body);
                if (params.isEmpty()) {
                    respond(exchange, 400, jsonObj("error", "需要1个参数"));
                    return;
                }
                int n = Integer.parseInt(params.get(0));
                StringBuilder fib = new StringBuilder("[");
                long a = 0, b = 1;
                for (int i = 0; i < n; i++) {
                    if (i > 0) fib.append(", ");
                    fib.append(a);
                    long temp = a;
                    a = b;
                    b = temp + b;
                }
                fib.append("]");
                respond(exchange, 200, jsonObj("n", String.valueOf(n), "fibonacci", fib.toString(), "lang", "Java"));
            } catch (Exception e) {
                respond(exchange, 500, jsonObj("error", e.getMessage(), "lang", "Java"));
            }
        }
    }

    static class FactorialHandler implements HttpHandler {
        public void handle(HttpExchange exchange) throws IOException {
            try {
                String body = readBody(exchange);
                List<String> params = parseParams(body);
                if (params.isEmpty()) {
                    respond(exchange, 400, jsonObj("error", "需要1个参数"));
                    return;
                }
                int n = Integer.parseInt(params.get(0));
                long result = 1;
                for (int i = 2; i <= n; i++) result *= i;
                respond(exchange, 200, jsonObj("n", String.valueOf(n), "factorial", String.valueOf(result), "lang", "Java"));
            } catch (Exception e) {
                respond(exchange, 500, jsonObj("error", e.getMessage(), "lang", "Java"));
            }
        }
    }
}
