/**
 * Jiasine CLI - JavaScript 进程调用测试
 * 通过子进程方式调用 (service type: process)
 *
 * 用法:
 *     node jiasine_js_process.js add 3 5
 *     node jiasine_js_process.js reverse hello
 *     node jiasine_js_process.js fibonacci 10
 */

const handlers = {
    add(params) {
        if (params.length < 2) return { error: '需要2个参数' };
        return { result: parseInt(params[0], 10) + parseInt(params[1], 10), lang: 'JavaScript' };
    },

    reverse(params) {
        if (params.length < 1) return { error: '需要1个参数' };
        const text = params[0];
        return { input: text, reversed: text.split('').reverse().join(''), lang: 'JavaScript' };
    },

    fibonacci(params) {
        if (params.length < 1) return { error: '需要1个参数' };
        const n = parseInt(params[0], 10);
        const fib = [];
        let a = 0, b = 1;
        for (let i = 0; i < n; i++) {
            fib.push(a);
            [a, b] = [b, a + b];
        }
        return { n, fibonacci: fib, lang: 'JavaScript' };
    }
};

function main() {
    const args = process.argv.slice(2);
    if (args.length < 1) {
        console.log(JSON.stringify({ error: '用法: node jiasine_js_process.js <method> [args...]' }));
        process.exit(1);
    }

    const method = args[0];
    const params = args.slice(1);

    const handler = handlers[method];
    if (!handler) {
        console.log(JSON.stringify({ error: `未知方法: ${method}`, available: Object.keys(handlers) }));
        process.exit(1);
    }

    const result = handler(params);
    console.log(JSON.stringify(result));
}

main();
