import json
import re

def extract_recent_files(text):
    """提取文本中的最近文件列表。"""
    # 查找常见的文件列表模式，例如 "Recent files:" 或 XML 标签
    # 在 Claude CLI 中通常表现为列表
    match = re.search(r"Recent files:?\n(.*?)(?:\n\n|\n[A-Z])", text, re.DOTALL | re.IGNORECASE)
    if match:
        return match.group(1).strip()
    
    # 尝试查找 git status 或类似输出中的文件
    if "Changes not staged for commit" in text or "Untracked files" in text:
        return "Found Git Status Output"
        
    return None

def analyze_recent_files():
    with open("sessions/change_claude_md.json", "r") as f:
        data = json.load(f)

    chat_calls = [d for d in data if "/v1/messages" in d["url"] and d.get("request_body") and not d["url"].endswith("count_tokens?beta=true")]
    
    print(f"分析 {len(chat_calls)} 条对话请求...\n")
    
    for i, call in enumerate(chat_calls):
        msgs = call["request_body"]["messages"]
        ts = call.get("timestamp")
        
        # 搜索所有消息块中的文件列表
        found_files = []
        for m in msgs:
            content = m["content"]
            if isinstance(content, list):
                for block in content:
                    if block.get("type") == "text":
                        extracted = extract_recent_files(block.get("text", ""))
                        if extracted: found_files.append(extracted)
            elif isinstance(content, str):
                extracted = extract_recent_files(content)
                if extracted: found_files.append(extracted)

        print(f"Trace #{i+1} [{ts}]:")
        if found_files:
            for f in found_files:
                print(f"  [FOUND] Recent Files:\n{f}")
        else:
            print("  [NOT FOUND] No recent files list detected.")
        print("-" * 40)

if __name__ == "__main__":
    analyze_recent_files()
