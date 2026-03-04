/**
 * Jiasine CLI - JavaScript 测试服务
 * 一个简单的 HTTP 服务，用于验证 CLI 的 Service 层调用 JavaScript 能力
 *
 * 用法:
 *     node jiasine_js_test.js              # 启动 HTTP 服务 (端口 9903)
 *     node jiasine_js_test.js --port 8080  # 指定端口
 *
 * 测试:
 *     curl http://localhost:9903/health
 *     curl -X POST http://localhost:9903/add -d '{"params":["3","5"]}'
 *     curl -X POST http://localhost:9903/reverse -d '{"params":["hello"]}'
 *     curl -X POST http://localhost:9903/fibonacci -d '{"params":["10"]}'
 */

const http = require('http');

// 解析命令行参数
function parseArgs() {
    const args = process.argv.slice(2);
    let port = 9903;
    let host = '127.0.0.1';
    for (let i = 0; i < args.length; i++) {
        if (args[i] === '--port' && args[i + 1]) {
            port = parseInt(args[i + 1], 10);
            i++;
        } else if (args[i] === '--host' && args[i + 1]) {
            host = args[i + 1];
            i++;
        }
    }
    return { port, host };
}

// 处理函数
const handlers = {
    add(params) {
        if (params.length < 2) throw new Error('需要至少2个参数');
        const a = parseInt(params[0], 10);
        const b = parseInt(params[1], 10);
        return { result: a + b, lang: 'JavaScript' };
    },

    reverse(params) {
        if (params.length < 1) throw new Error('需要至少1个参数');
        const text = params[0];
        return { input: text, reversed: text.split('').reverse().join(''), lang: 'JavaScript' };
    },

    fibonacci(params) {
        if (params.length < 1) throw new Error('需要1个参数 (数列长度)');
        const n = parseInt(params[0], 10);
        const fib = [];
        let a = 0, b = 1;
        for (let i = 0; i < n; i++) {
            fib.push(a);
            [a, b] = [b, a + b];
        }
        return { n, fibonacci: fib, lang: 'JavaScript' };
    },

    upper(params) {
        if (params.length < 1) throw new Error('需要至少1个参数');
        const text = params.join(' ');
        return { input: text, upper: text.toUpperCase(), lang: 'JavaScript' };
    },

    factorial(params) {
        if (params.length < 1) throw new Error('需要1个参数');
        const n = parseInt(params[0], 10);
        let result = 1;
        for (let i = 2; i <= n; i++) result *= i;
        return { n, factorial: result, lang: 'JavaScript' };
    }
};

// 读取请求体
function readBody(req) {
    return new Promise((resolve, reject) => {
        let body = '';
        req.on('data', chunk => { body += chunk; });
        req.on('end', () => resolve(body));
        req.on('error', reject);
    });
}

// JSON 响应
function respond(res, statusCode, data) {
    res.writeHead(statusCode, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify(data));
}

// 创建服务
const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, `http://${req.headers.host}`);
    const path = url.pathname.replace(/^\/+|\/+$/g, '');

    if (req.method === 'GET') {
        if (path === 'health') {
            return respond(res, 200, { status: 'ok', lang: 'JavaScript', version: process.version });
        }
        if (path === 'version') {
            return respond(res, 200, {
                name: 'jiasine_js_test',
                version: '1.0.0',
                lang: 'JavaScript',
                node_version: process.version
            });
        }
        return respond(res, 404, { error: `未知路径: /${path}` });
    }

    if (req.method === 'POST') {
        try {
            const body = await readBody(req);
            const data = body ? JSON.parse(body) : {};
            const params = data.params || [];

            const handler = handlers[path];
            if (!handler) {
                return respond(res, 404, { error: `未知方法: ${path}`, available: Object.keys(handlers) });
            }

            const result = handler(params);
            return respond(res, 200, result);
        } catch (e) {
            return respond(res, 500, { error: e.message, lang: 'JavaScript' });
        }
    }

    respond(res, 405, { error: 'Method not allowed' });
});

const { port, host } = parseArgs();
server.listen(port, host, () => {
    console.log(`[JS Service] 启动于 http://${host}:${port}`);
});
