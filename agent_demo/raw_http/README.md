# Qwen HTTP Demo (Go)

用 Go 标准库 `net/http` 直接调用阿里云 DashScope 的 Qwen（OpenAI 兼容接口）示例。

## 运行

1) 设置 API Key（DashScope 控制台创建）

```bash
export QWEN_API_KEY="YOUR_KEY"
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

## 渲染对话过程

- `Client.Invoke(...)` 返回 `*InvokeResult`，可用 `RenderInvokeResult(invoke)` 生成一段可读的对话输出。
- `doReACT(...)` 返回 `*ReACTResult`（包含 `Messages` / `Invokes`），可用 `RenderReACTResult(res)` 渲染完整的调用历史（含 tool_calls 与 tool 输出）。
- 如果模型返回 `reasoning_content`（或输出了 `<think>...</think>` / `<final>...</final>`），渲染器会把 think 与最终答案分开展示；可用 `WithThink(systemPrompt)` 给 system prompt 追加一段约束格式的指令。
