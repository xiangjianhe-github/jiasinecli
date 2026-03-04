/**
 * Jiasine CLI - TypeScript 测试服务
 * 一个简单的 HTTP 服务，用于验证 CLI 的 Service 层调用 TypeScript 能力
 *
 * 用法:
 *     npx tsx jiasine_ts_test.ts              # 启动 HTTP 服务 (端口 9904)
 *     npx tsx jiasine_ts_test.ts --port 8080  # 指定端口
 *
 * 测试:
 *     curl http://localhost:9904/health
 *     curl -X POST http://localhost:9904/add -d '{"params":["3","5"]}'
 */

import * as http from 'http';

// 类型定义
interface RequestData {
    params: string[];
}

interface ResponseData {
    [key: string]: unknown;
}

type HandlerFn = (params: string[]) => ResponseData;

// 解析命令行参数
function parseArgs(): { port: number; host: string } {
    const args = process.argv.slice(2);
    let port = 9904;
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

// 处理函数 (强类型)
const handlers: Record<string, HandlerFn> = {
    add(params: string[]): ResponseData {
        if (params.length < 2) throw new Error('需要至少2个参数');
        const a: number = parseInt(params[0], 10);
        const b: number = parseInt(params[1], 10);
        return { result: a + b, lang: 'TypeScript' };
    },

    reverse(params: string[]): ResponseData {
        if (params.length < 1) throw new Error('需要至少1个参数');
        const text: string = params[0];
        return { input: text, reversed: text.split('').reverse().join(''), lang: 'TypeScript' };
    },

    fibonacci(params: string[]): ResponseData {
        if (params.length < 1) throw new Error('需要1个参数 (数列长度)');
        const n: number = parseInt(params[0], 10);
        const fib: number[] = [];
        let a = 0, b = 1;
        for (let i = 0; i < n; i++) {
            fib.push(a);
            [a, b] = [b, a + b];
        }
        return { n, fibonacci: fib, lang: 'TypeScript' };
    },

    upper(params: string[]): ResponseData {
        if (params.length < 1) throw new Error('需要至少1个参数');
        const text: string = params.join(' ');
        return { input: text, upper: text.toUpperCase(), lang: 'TypeScript' };
    },

    factorial(params: string[]): ResponseData {
        if (params.length < 1) throw new Error('需要1个参数');
        const n: number = parseInt(params[0], 10);
        let result = 1;
        for (let i = 2; i <= n; i++) result *= i;
        return { n, factorial: result, lang: 'TypeScript' };
    }
};

// 读取请求体
function readBody(req: http.IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
        let body = '';
        req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
        req.on('end', () => resolve(body));
        req.on('error', reject);
    });
}

// JSON 响应
function respond(res: http.ServerResponse, statusCode: number, data: ResponseData): void {
    res.writeHead(statusCode, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify(data));
}

// 创建服务
const server = http.createServer(async (req: http.IncomingMessage, res: http.ServerResponse) => {
    const url = new URL(req.url || '/', `http://${req.headers.host}`);
    const path = url.pathname.replace(/^\/+|\/+$/g, '');

    if (req.method === 'GET') {
        if (path === 'health') {
            return respond(res, 200, { status: 'ok', lang: 'TypeScript', version: process.version });
        }
        if (path === 'version') {
            return respond(res, 200, {
                name: 'jiasine_ts_test',
                version: '1.0.0',
                lang: 'TypeScript',
                node_version: process.version
            });
        }
        return respond(res, 404, { error: `未知路径: /${path}` });
    }

    if (req.method === 'POST') {
        try {
            const body = await readBody(req);
            const data: RequestData = body ? JSON.parse(body) : { params: [] };
            const params: string[] = data.params || [];

            const handler = handlers[path];
            if (!handler) {
                return respond(res, 404, { error: `未知方法: ${path}`, available: Object.keys(handlers) });
            }

            const result = handler(params);
            return respond(res, 200, result);
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : String(e);
            return respond(res, 500, { error: msg, lang: 'TypeScript' });
        }
    }

    respond(res, 405, { error: 'Method not allowed' });
});

const { port, host } = parseArgs();
server.listen(port, host, () => {
    console.log(`[TS Service] 启动于 http://${host}:${port}`);
});
