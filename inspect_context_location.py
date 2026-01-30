import json

def analyze():
    with open("sessions/mcp.json", "r") as f:
        data = json.load(f)

    # Find the trace with 5 messages (a later turn)
    trace = next((d for d in data if "/v1/messages" in d["url"] and len(d.get("request_body", {}).get("messages", [])) == 5), None)

    if not trace:
        print("Trace not found.")
        return

    body = trace["request_body"]
    
    print("=== 1. SYSTEM PROMPT ===")
    system = body.get("system", [])
    if isinstance(system, list):
        for i, s in enumerate(system):
            text = s.get("text", "")
            print(f"Block {i+1}: {text[:60]}...")
            if "CLAUDE.md" in text:
                print("  -> [FOUND] CLAUDE.md content is here!")
            if "cache_control" in s:
                print(f"  -> [CACHE] cache_control present: {s['cache_control']}")

    print("\n=== 2. MESSAGE HISTORY ===")
    messages = body["messages"]
    for i, msg in enumerate(messages):
        role = msg["role"]
        content = msg["content"]
        print(f"Msg {i+1} ({role}):")
        
        if isinstance(content, list):
            for j, block in enumerate(content):
                text = block.get("text", "") if block.get("type") == "text" else str(block)[:50]
                print(f"  Block {j+1} [{block.get('type')}]: {text[:60]}...")
                
                # Check for dynamic indicators
                if "CLAUDE.md" in text:
                    print("    -> [FOUND] CLAUDE.md content is here!")
                if "<local-command-stdout>" in text:
                    print("    -> [FOUND] Command Output is here!")
                
                if "cache_control" in block:
                    print(f"    -> [CACHE] cache_control present: {block['cache_control']}")
        else:
            print(f"  Content: {content[:50]}...")

if __name__ == "__main__":
    analyze()
