import json
import sys

def clean_msg(msg):
    """Strip cache_control and other ephemeral fields for comparison."""
    return {
        "role": msg.get("role"),
        "content": msg.get("content")
    }

def msgs_equal(msg1, msg2):
    c1 = clean_msg(msg1)
    c2 = clean_msg(msg2)
    
    # Deep comparison
    if c1['role'] != c2['role']:
        return False, f"Role mismatch: {c1['role']} != {c2['role']}"
    
    # Content comparison (handle lists vs strings)
    # This is where it gets tricky if JSON serialization order differs, 
    # but Python dict comparison handles order insensitivity.
    if c1['content'] != c2['content']:
        # Let's try to be more specific if it's a list
        return False, "Content mismatch"
        
    return True, "Match"

def run_test():
    with open("sessions/mcp.json", "r") as f:
        data = json.load(f)

    # Filter calls exactly like the frontend
    chat_calls = [
        d for d in data 
        if "/v1/messages" in d["url"] 
        and d.get("request_body") 
        and not d["url"].endswith("count_tokens?beta=true")
        and d.get("request_body", {}).get("messages")
    ]
    
    # Sort by timestamp
    chat_calls.sort(key=lambda x: x.get("timestamp", 0))
    
    print(f"Loaded {len(chat_calls)} chat calls.")
    
    branches = [] # List of lists of calls
    
    for i, call in enumerate(chat_calls):
        current_msgs = call["request_body"]["messages"]
        print(f"\nProcessing Trace #{i+1} (Length: {len(current_msgs)})")
        
        found_branch = False
        
        # Iterate backwards to find most recent branch
        for b_idx in range(len(branches) - 1, -1, -1):
            branch = branches[b_idx]
            last_call = branch[-1]
            last_msgs = last_call["request_body"]["messages"]
            
            print(f"  Checking vs Branch {b_idx+1} Tip (Length: {len(last_msgs)})...")
            
            if len(current_msgs) >= len(last_msgs):
                # Check prefix
                is_prefix = True
                fail_reason = ""
                
                for k in range(len(last_msgs)):
                    m1 = last_msgs[k]
                    m2 = current_msgs[k]
                    is_match, reason = msgs_equal(m1, m2)
                    if not is_match:
                        is_prefix = False
                        fail_reason = f"Msg[{k}] mismatch: {reason}"
                        # Detailed debug
                        print(f"    [DEBUG] Mismatch at index {k}:")
                        print(f"      Branch: {clean_msg(m1)}")
                        print(f"      Trace : {clean_msg(m2)}")
                        break
                
                if is_prefix:
                    print(f"    -> MATCH! Extending Branch {b_idx+1}")
                    branch.append(call)
                    found_branch = True
                    break
                else:
                    print(f"    -> No match. {fail_reason}")
            else:
                print(f"    -> No match. Trace shorter than branch tip.")
        
        if not found_branch:
            print(f"  -> Creating NEW Branch {len(branches)+1}")
            branches.append([call])

    print("\n" + "="*40)
    print("FINAL BRANCH STRUCTURE")
    print("="*40)
    for i, branch in enumerate(branches):
        print(f"Branch {i+1}: {len(branch)} calls")
        for j, call in enumerate(branch):
            msgs = call["request_body"]["messages"]
            preview = msgs[-1]["content"] if msgs else "Empty"
            if isinstance(preview, list): preview = str(preview)[:30]
            elif isinstance(preview, str): preview = preview[:30]
            print(f"  - Call {j+1}: {len(msgs)} msgs -> {preview}...")

if __name__ == "__main__":
    run_test()
