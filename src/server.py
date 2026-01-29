import http.server
import socketserver
import json
import os
import mimetypes

PORT = 8000
WEB_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "web")
SESSIONS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "sessions")

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        # API: List sessions
        if self.path == "/api/sessions":
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            files = [f for f in os.listdir(SESSIONS_DIR) if f.endswith(".json")]
            files.sort(key=lambda x: os.path.getmtime(os.path.join(SESSIONS_DIR, x)), reverse=True)
            self.wfile.write(json.dumps(files).encode("utf-8"))
            return

        # API: Get session content
        if self.path.startswith("/api/sessions/"):
            filename = self.path.replace("/api/sessions/", "")
            # Basic security check
            if "/" in filename or "\" in filename:
                self.send_error(400, "Invalid filename")
                return
            
            filepath = os.path.join(SESSIONS_DIR, filename)
            if os.path.exists(filepath):
                self.send_response(200)
                self.send_header("Content-type", "application/json")
                self.end_headers()
                with open(filepath, "rb") as f:
                    self.wfile.write(f.read())
            else:
                self.send_error(404, "File not found")
            return

        # Serve static files from web/ directory
        # Map root / to index.html
        if self.path == "/" or self.path == "/index.html":
            filepath = os.path.join(WEB_DIR, "index.html")
            self.serve_file(filepath)
            return

        # Serve other static files if any
        filepath = os.path.join(WEB_DIR, self.path.lstrip("/"))
        if os.path.exists(filepath) and os.path.isfile(filepath):
            self.serve_file(filepath)
        else:
            self.send_error(404, "File not found")

    def serve_file(self, filepath):
        self.send_response(200)
        mime_type, _ = mimetypes.guess_type(filepath)
        if mime_type:
            self.send_header("Content-type", mime_type)
        self.end_headers()
        with open(filepath, "rb") as f:
            self.wfile.write(f.read())

print(f"Starting viewer at http://localhost:{PORT}")
print(f"Web root: {WEB_DIR}")
print(f"Sessions root: {SESSIONS_DIR}")

with socketserver.TCPServer(("", PORT), Handler) as httpd:
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    print("\nServer stopped.")
