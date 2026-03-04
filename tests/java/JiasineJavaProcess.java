/**
 * Jiasine CLI - Java 进程调用测试
 * 通过子进程方式调用 (service type: process)
 *
 * 编译: javac JiasineJavaProcess.java
 * 用法:
 *     java JiasineJavaProcess add 3 5
 *     java JiasineJavaProcess reverse hello
 *     java JiasineJavaProcess fibonacci 10
 */

public class JiasineJavaProcess {

    public static void main(String[] args) {
        if (args.length < 1) {
            System.out.println("{\"error\": \"用法: java JiasineJavaProcess <method> [args...]\"}");
            System.exit(1);
        }

        String method = args[0];
        String[] params = new String[args.length - 1];
        System.arraycopy(args, 1, params, 0, params.length);

        String result;
        switch (method) {
            case "add":
                result = handleAdd(params);
                break;
            case "reverse":
                result = handleReverse(params);
                break;
            case "fibonacci":
                result = handleFibonacci(params);
                break;
            default:
                result = "{\"error\": \"未知方法: " + method + "\", \"available\": [\"add\", \"reverse\", \"fibonacci\"]}";
                break;
        }

        System.out.println(result);
    }

    static String handleAdd(String[] params) {
        if (params.length < 2) return "{\"error\": \"需要2个参数\"}";
        int a = Integer.parseInt(params[0]);
        int b = Integer.parseInt(params[1]);
        return "{\"result\": " + (a + b) + ", \"lang\": \"Java\"}";
    }

    static String handleReverse(String[] params) {
        if (params.length < 1) return "{\"error\": \"需要1个参数\"}";
        String text = params[0];
        String reversed = new StringBuilder(text).reverse().toString();
        return "{\"input\": \"" + text + "\", \"reversed\": \"" + reversed + "\", \"lang\": \"Java\"}";
    }

    static String handleFibonacci(String[] params) {
        if (params.length < 1) return "{\"error\": \"需要1个参数\"}";
        int n = Integer.parseInt(params[0]);
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
        return "{\"n\": " + n + ", \"fibonacci\": " + fib + ", \"lang\": \"Java\"}";
    }
}
