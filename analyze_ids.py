import json

with open("sessions/mcp.json", "r") as f:
    data = json.load(f)

# 过滤出对话请求
chat_calls = [d for d in data if "/v1/messages" in d["url"] and d.get("request_body") and not d["url"].endswith("count_tokens?beta=true")]

print(f"Total Chat Traces: {len(chat_calls)}\n")

for i, trace in enumerate(chat_calls):
    time = trace.get("timestamp")
    # 打印时间、消息数、以及一些可能作为特征的字段
    headers = trace.get("request_headers", {})
    body = trace.get("request_body", {})
    
    # 查找可能的 ID：
    # 1. metadata 里的隐藏字段
    # 2. headers 里的自定义字段 (x-...)
    # 3. system prompt 里的特征字符串
    
    billing = ""
    system = body.get("system", "")
    if isinstance(system, list):
        for s in system:
            if "billing-header" in s.get("text", ""):
                billing = s.get("text", "").split(";")[0] # cc_version=...
    
    msg_preview = ""
    msgs = body.get("messages", [])
    if msgs:
        last = msgs[-1].get("content", "")
        msg_preview = str(last)[:40].replace("\n", " ")

    print(f"#{i+1} [{time}] msgs:{len(msgs)} | {msg_preview}")
    # 打印有趣的 Header
    interesting_headers = {k: v for k, v in headers.items() if k.lower().startswith('x-')}
    print(f"  Headers: {interesting_headers}")
    # 打印 Metadata
    print(f"  Metadata: {body.get('metadata')}")
    print("-" * 40)
