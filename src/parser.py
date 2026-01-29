import json
from mitmproxy import http

class Extractor:
    def __init__(self):
        self.calls = []

    def response(self, flow: http.HTTPFlow):
        # Focus on the messages API endpoint
        if "/v1/messages" in flow.request.pretty_url and flow.request.method == "POST":
            call_data = {
                "timestamp": flow.request.timestamp_start,
                "url": flow.request.pretty_url,
                "request_headers": dict(flow.request.headers),
                "response_headers": dict(flow.response.headers),
                "request_body": None,
                "response_body": None
            }

            # Parse Request Body
            try:
                if flow.request.content:
                    call_data["request_body"] = json.loads(flow.request.content)
            except Exception as e:
                call_data["request_body"] = f"<Error parsing JSON: {str(e)}>"

            # Parse Response Body
            try:
                content_type = flow.response.headers.get("Content-Type", "")
                if "text/event-stream" in content_type and flow.response.content:
                    # Handle SSE stream
                    full_text = ""
                    raw_content = flow.response.content.decode("utf-8", errors="replace")
                    
                    # Simple SSE parser
                    for line in raw_content.splitlines():
                        if line.startswith("data: ") and line != "data: [DONE]":
                            try:
                                data = json.loads(line[6:])
                                if data.get("type") == "content_block_delta":
                                    delta = data.get("delta", {})
                                    if delta.get("type") == "text_delta":
                                        full_text += delta.get("text", "")
                            except:
                                pass
                    
                    if full_text:
                        call_data["response_body"] = {"reconstructed_text": full_text}
                    else:
                         call_data["response_body"] = "<SSE Stream (parsed but no text content found)>"

                elif flow.response.content:
                    call_data["response_body"] = json.loads(flow.response.content)
            except Exception as e:
                call_data["response_body"] = f"<Error parsing JSON: {str(e)}>"

            self.calls.append(call_data)

    def done(self):
        import os
        output_file = os.environ.get("OUTPUT_JSON_PATH", "parsed_llm_calls.json")
        try:
            with open(output_file, "w", encoding="utf-8") as f:
                json.dump(self.calls, f, indent=2, ensure_ascii=False)
            print(f"Successfully wrote {len(self.calls)} LLM calls to {output_file}")
        except Exception as e:
            print(f"Error writing output file: {e}")

addons = [Extractor()]
