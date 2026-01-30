# Qwen HTTP Demo (Go)

用 Go 标准库 `net/http` 直接调用阿里云 DashScope 的 Qwen（OpenAI 兼容接口）示例。

## 运行

1) 设置 API Key（DashScope 控制台创建）

```bash
export DASHSCOPE_API_KEY="YOUR_KEY"
```

也支持：

```bash
export ALIYUN_DASHSCOPE_API_KEY="YOUR_KEY"
```

2) 运行

```bash
go run . -prompt "用一句话介绍一下Qwen"
```

## 工具调用（Tool Calling）/ ReACT

内置了两个工具：

- `now`：获取当前时间（支持 `timezone` / `format`）
- `calculator`：计算简单四则运算表达式

开启 ReACT 循环（自动处理 tool_calls）：

```bash
go run . -react -prompt "请调用 now 工具给出 Asia/Shanghai 当前时间，然后用 calculator 计算 (1+2)*3"
```

常用参数：

- `-model`：默认 `qwen-plus`（也可改成 `qwen-turbo` 等）
- `-endpoint`：默认 `https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions`
- `-system`：system prompt
- `-temp`：temperature
- `-timeout`：请求超时
- `-react`：开启 ReACT 循环
- `-max-steps`：ReACT 最大步数
