/**
 * Jiasine CLI - TypeScript 进程调用测试
 * 通过子进程方式调用 (service type: process)
 *
 * 用法:
 *     npx tsx jiasine_ts_process.ts add 3 5
 *     npx tsx jiasine_ts_process.ts reverse hello
 *     npx tsx jiasine_ts_process.ts fibonacci 10
 */

interface ResultData {
    [key: string]: unknown;
}

type HandlerFn = (params: string[]) => ResultData;

const handlers: Record<string, HandlerFn> = {
    add(params: string[]): ResultData {
        if (params.length < 2) return { error: '需要2个参数' };
        return { result: parseInt(params[0], 10) + parseInt(params[1], 10), lang: 'TypeScript' };
    },

    reverse(params: string[]): ResultData {
        if (params.length < 1) return { error: '需要1个参数' };
        const text: string = params[0];
        return { input: text, reversed: text.split('').reverse().join(''), lang: 'TypeScript' };
    },

    fibonacci(params: string[]): ResultData {
        if (params.length < 1) return { error: '需要1个参数' };
        const n: number = parseInt(params[0], 10);
        const fib: number[] = [];
        let a = 0, b = 1;
        for (let i = 0; i < n; i++) {
            fib.push(a);
            [a, b] = [b, a + b];
        }
        return { n, fibonacci: fib, lang: 'TypeScript' };
    }
};

function main(): void {
    const args: string[] = process.argv.slice(2);
    if (args.length < 1) {
        console.log(JSON.stringify({ error: '用法: npx tsx jiasine_ts_process.ts <method> [args...]' }));
        process.exit(1);
    }

    const method: string = args[0];
    const params: string[] = args.slice(1);

    const handler = handlers[method];
    if (!handler) {
        console.log(JSON.stringify({ error: `未知方法: ${method}`, available: Object.keys(handlers) }));
        process.exit(1);
    }

    const result = handler(params);
    console.log(JSON.stringify(result));
}

main();
