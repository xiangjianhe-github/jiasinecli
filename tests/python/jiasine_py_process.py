"""
Jiasine CLI - Python 进程调用测试
通过子进程方式调用 (service type: process)

用法:
    python jiasine_py_process.py add 3 5
    python jiasine_py_process.py reverse hello
    python jiasine_py_process.py fibonacci 10
"""

import json
import sys


def add(params):
    if len(params) < 2:
        return {"error": "需要2个参数"}
    return {"result": int(params[0]) + int(params[1]), "lang": "Python"}


def reverse(params):
    if not params:
        return {"error": "需要1个参数"}
    text = params[0]
    return {"input": text, "reversed": text[::-1], "lang": "Python"}


def fibonacci(params):
    if not params:
        return {"error": "需要1个参数"}
    n = int(params[0])
    fib = []
    a, b = 0, 1
    for _ in range(n):
        fib.append(a)
        a, b = b, a + b
    return {"n": n, "fibonacci": fib, "lang": "Python"}


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "用法: python jiasine_py_process.py <method> [args...]"}))
        sys.exit(1)

    method = sys.argv[1]
    params = sys.argv[2:]

    handlers = {
        "add": add,
        "reverse": reverse,
        "fibonacci": fibonacci,
    }

    handler = handlers.get(method)
    if handler is None:
        print(json.dumps({"error": f"未知方法: {method}", "available": list(handlers.keys())}))
        sys.exit(1)

    result = handler(params)
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
