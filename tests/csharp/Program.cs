/*
 * Jiasine CLI - C# 测试服务 (ASP.NET Core Minimal API)
 *
 * 创建与运行:
 *   dotnet new web -o . --force  (如需初始化)
 *   dotnet run --urls http://localhost:9902
 *
 * 发布为 Native AOT (生成动态库):
 *   dotnet publish -c Release -r win-x64 /p:PublishAot=true
 *   dotnet publish -c Release -r linux-x64 /p:PublishAot=true
 */

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

// 健康检查
app.MapGet("/health", () => Results.Json(new
{
    status = "ok",
    lang = "C#",
    dotnet = Environment.Version.ToString()
}));

// 版本信息
app.MapGet("/version", () => Results.Json(new
{
    name = "jiasine_csharp_test",
    version = "1.0.0",
    lang = "C#",
    dotnet_version = Environment.Version.ToString(),
    os = Environment.OSVersion.ToString()
}));

// 加法
app.MapPost("/add", (RequestBody body) =>
{
    var p = body.Params ?? Array.Empty<string>();
    if (p.Length < 2) return Results.BadRequest(new { error = "需要至少2个参数" });

    int a = int.Parse(p[0]);
    int b = int.Parse(p[1]);
    return Results.Json(new { result = a + b, lang = "C#" });
});

// 字符串反转
app.MapPost("/reverse", (RequestBody body) =>
{
    var p = body.Params ?? Array.Empty<string>();
    if (p.Length < 1) return Results.BadRequest(new { error = "需要至少1个参数" });

    string input = p[0];
    char[] chars = input.ToCharArray();
    Array.Reverse(chars);
    string reversed = new string(chars);
    return Results.Json(new { input, reversed, lang = "C#" });
});

// 阶乘
app.MapPost("/factorial", (RequestBody body) =>
{
    var p = body.Params ?? Array.Empty<string>();
    if (p.Length < 1) return Results.BadRequest(new { error = "需要1个参数" });

    int n = int.Parse(p[0]);
    long result = 1;
    for (int i = 2; i <= n; i++) result *= i;
    return Results.Json(new { n, factorial = result, lang = "C#" });
});

Console.WriteLine("[Jiasine C# Test Service] 启动中...");
app.Run();

record RequestBody(string? Method, string[]? Params);
