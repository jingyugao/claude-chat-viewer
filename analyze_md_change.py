import json
import re

def extract_claude_md(text):
    """Extract CLAUDE.md content from a text block."""
    if "Contents of" not in text and "CLAUDE.md" not in text:
        return None
    
    # Rough extraction logic based on previous observations
    match = re.search(r"Contents of .*CLAUDE\.md.*:\n\n(.*?)\n\n\s*IMPORTANT", text, re.DOTALL)
    if match:
        return match.group(1).strip()
    return None

def analyze_claude_md_change():
    try:
        with open("sessions/change_claude_md.json", "r") as f:
            data = json.load(f)
    except FileNotFoundError:
        print("File not found.")
        return

    chat_calls = [d for d in data if "/v1/messages" in d["url"] and d.get("request_body") and not d["url"].endswith("count_tokens?beta=true")]
    
    print(f"Found {len(chat_calls)} chat calls.")
    
    previous_md = None
    
    for i, call in enumerate(chat_calls):
        msgs = call["request_body"]["messages"]
        if not msgs: continue
        
        # Usually CLAUDE.md is in the first user message
        first_msg = msgs[0]
        claude_md_content = "Not Found"
        
        if first_msg["role"] == "user":
            content = first_msg["content"]
            if isinstance(content, list):
                for block in content:
                    if block.get("type") == "text":
                        extracted = extract_claude_md(block.get("text", ""))
                        if extracted:
                            claude_md_content = extracted
                            break
            elif isinstance(content, str):
                extracted = extract_claude_md(content)
                if extracted:
                    claude_md_content = extracted

        print(f"\nTrace #{i+1} (Msg Count: {len(msgs)})")
        print(f"  CLAUDE.md Content: {claude_md_content[:50]}...")
        
        if previous_md is not None and claude_md_content != "Not Found":
            if claude_md_content != previous_md:
                print("  [! ] CHANGE DETECTED compared to previous trace!")
                print("  Since CLAUDE.md changed in Msg[0], the prefix cache is BROKEN.")
            else:
                print("  [=] No change in CLAUDE.md content.")
        
        if claude_md_content != "Not Found":
            previous_md = claude_md_content

if __name__ == "__main__":
    analyze_claude_md_change()
