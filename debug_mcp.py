import json

with open("sessions/mcp.json", "r") as f:
    data = json.load(f)

print(f"Total calls: {len(data)}")

chat_calls = [d for d in data if "/v1/messages" in d["url"] and d.get("request_body")]

print(f"Chat calls: {len(chat_calls)}")

for i, call in enumerate(chat_calls):
    msgs = call["request_body"].get("messages", [])
    metadata = call["request_body"].get("metadata", {})
    print(f"Call {i+1}: {call['url']} - {len(msgs)} msgs")
    print(f"  Metadata: {metadata}")
    
    # Print the last message in the request to see what it is
    if msgs:
        last_role = msgs[-1]["role"]
        content = msgs[-1]["content"]
        if isinstance(content, list):
            content_summary = str(content)[:50] + "..."
        else:
            content_summary = content[:50] + "..."
        print(f"  Last Req Msg: {last_role} - {content_summary}")
