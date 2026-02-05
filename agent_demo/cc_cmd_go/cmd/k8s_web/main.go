package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gao/claude-chat-viewer/agent_demo/cc_cmd_go/pkg/k8sagent"
)

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
}

type session struct {
	Messages []k8sagent.ChatMessage
	SeenAt   time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*session
}

func main() {
	var addr string
	var kubeconfig string
	var kubeContext string
	var namespace string
	var timeout time.Duration
	var model string
	var tools string
	var systemPrompt string
	var mode string

	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	flag.StringVar(&kubeContext, "context", "", "Kube context")
	flag.StringVar(&namespace, "namespace", "", "Namespace filter (default all)")
	flag.DurationVar(&timeout, "timeout", 20*time.Second, "kubectl timeout")
	flag.StringVar(&model, "model", "", "Claude model name (optional)")
	flag.StringVar(&tools, "tools", "", "Claude --tools value (optional)")
	flag.StringVar(&systemPrompt, "system", "", "Extra system prompt to append")
	flag.StringVar(&mode, "mode", "summary", "Agent mode: summary or bash")
	flag.Parse()

	agent := k8sagent.New(k8sagent.Config{
		Kubeconfig:   kubeconfig,
		Context:      kubeContext,
		Namespace:    namespace,
		Timeout:      timeout,
		Model:        model,
		Tools:        tools,
		SystemPrompt: systemPrompt,
		Mode:         k8sagent.Mode(mode),
	})

	store := &sessionStore{sessions: make(map[string]*session)}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ensureSession(store, w, r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		sid := ensureSession(store, w, r)
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, chatResponse{Error: "invalid request"})
			return
		}
		question := strings.TrimSpace(req.Message)
		if question == "" {
			writeJSON(w, chatResponse{Error: "message is empty"})
			return
		}

		history := store.getHistory(sid)
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		reply, err := agent.AskWithHistory(ctx, question, history)
		if err != nil {
			writeJSON(w, chatResponse{Error: err.Error()})
			return
		}
		store.appendMessage(sid, k8sagent.ChatMessage{Role: "user", Content: question})
		store.appendMessage(sid, k8sagent.ChatMessage{Role: "assistant", Content: reply})

		writeJSON(w, chatResponse{Reply: reply})
	})

	mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		sid := ensureSession(store, w, r)
		store.reset(sid)
		writeJSON(w, chatResponse{Reply: "ok"})
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("k8s web agent listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func ensureSession(store *sessionStore, w http.ResponseWriter, r *http.Request) string {
	cookie, _ := r.Cookie("sid")
	if cookie != nil && cookie.Value != "" {
		store.touch(cookie.Value)
		return cookie.Value
	}
	sid := newSessionID()
	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   24 * 3600,
	})
	store.touch(sid)
	return sid
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *sessionStore) touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[id] == nil {
		s.sessions[id] = &session{SeenAt: time.Now()}
		return
	}
	s.sessions[id].SeenAt = time.Now()
}

func (s *sessionStore) getHistory(id string) []k8sagent.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[id] == nil {
		return nil
	}
	h := s.sessions[id].Messages
	out := make([]k8sagent.ChatMessage, len(h))
	copy(out, h)
	return out
}

func (s *sessionStore) appendMessage(id string, msg k8sagent.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[id] == nil {
		s.sessions[id] = &session{SeenAt: time.Now()}
	}
	s.sessions[id].Messages = append(s.sessions[id].Messages, msg)
	s.sessions[id].SeenAt = time.Now()
}

func (s *sessionStore) reset(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[id] != nil {
		s.sessions[id].Messages = nil
		s.sessions[id].SeenAt = time.Now()
	}
}

func writeJSON(w http.ResponseWriter, resp chatResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

const indexHTML = `<!doctype html>
<html lang="zh">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>K8s Agent Chat</title>
  <style>
    :root {
      --bg: #0b0f14;
      --panel: #121821;
      --border: #223041;
      --text: #e6edf3;
      --muted: #9db2c3;
      --accent: #5cc8ff;
      --user: #1e2a38;
      --assistant: #14222f;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Iosevka Term", "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      background: radial-gradient(1200px 800px at 10% -10%, #233246 0%, #0b0f14 55%), #0b0f14;
      color: var(--text);
    }
    .wrap {
      max-width: 980px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 20px;
    }
    .title {
      font-size: 20px;
      letter-spacing: 0.04em;
      text-transform: uppercase;
    }
    .badge {
      padding: 6px 10px;
      background: #16202b;
      border: 1px solid var(--border);
      color: var(--muted);
      font-size: 12px;
      border-radius: 999px;
    }
    .panel {
      background: linear-gradient(180deg, rgba(23,31,43,0.9), rgba(16,24,33,0.9));
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 16px;
      box-shadow: 0 10px 30px rgba(0,0,0,0.25);
    }
    .log {
      height: 480px;
      overflow: auto;
      padding: 6px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .msg {
      padding: 12px 14px;
      border-radius: 10px;
      line-height: 1.5;
      white-space: pre-wrap;
      border: 1px solid transparent;
    }
    .msg.user {
      background: var(--user);
      border-color: #2b3b4d;
      align-self: flex-end;
    }
    .msg.assistant {
      background: var(--assistant);
      border-color: #1f3245;
      align-self: flex-start;
    }
    .muted { color: var(--muted); }

    form {
      margin-top: 16px;
      display: flex;
      gap: 10px;
    }
    input[type="text"] {
      flex: 1;
      padding: 12px 14px;
      border-radius: 10px;
      border: 1px solid var(--border);
      background: #0e141c;
      color: var(--text);
    }
    button {
      padding: 12px 16px;
      border-radius: 10px;
      border: 1px solid var(--border);
      background: #17212b;
      color: var(--text);
      cursor: pointer;
    }
    button.primary {
      background: linear-gradient(135deg, #1b87d1, #5cc8ff);
      border-color: transparent;
      color: #07131f;
      font-weight: 600;
    }
    button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="header">
      <div class="title">K8s Agent Chat</div>
      <div class="badge">kubectl + claude</div>
    </div>
    <div class="panel">
      <div id="log" class="log">
        <div class="msg assistant">你好，我可以帮你分析集群 Service 信息。你可以问：
- 哪些 LoadBalancer 还没有外网 IP？
- service 类型分布如何？
- 哪些服务没有 selector？</div>
      </div>
      <form id="chat-form">
        <input id="chat-input" type="text" placeholder="输入你的问题..." autocomplete="off" />
        <button class="primary" id="send-btn" type="submit">发送</button>
        <button id="reset-btn" type="button">重置</button>
      </form>
      <div class="muted" id="status"></div>
    </div>
  </div>

<script>
const log = document.getElementById('log');
const form = document.getElementById('chat-form');
const input = document.getElementById('chat-input');
const sendBtn = document.getElementById('send-btn');
const resetBtn = document.getElementById('reset-btn');
const statusEl = document.getElementById('status');

function append(role, text) {
  const div = document.createElement('div');
  div.className = 'msg ' + role;
  div.textContent = text;
  log.appendChild(div);
  log.scrollTop = log.scrollHeight;
}

async function sendMessage(message) {
  statusEl.textContent = '处理中...';
  sendBtn.disabled = true;
  input.disabled = true;
  try {
    const resp = await fetch('/api/chat', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({message})
    });
    const data = await resp.json();
    if (data.error) {
      append('assistant', '错误: ' + data.error);
    } else {
      append('assistant', data.reply || '(empty)');
    }
  } catch (e) {
    append('assistant', '请求失败: ' + e);
  } finally {
    statusEl.textContent = '';
    sendBtn.disabled = false;
    input.disabled = false;
    input.focus();
  }
}

form.addEventListener('submit', (e) => {
  e.preventDefault();
  const message = input.value.trim();
  if (!message) return;
  append('user', message);
  input.value = '';
  sendMessage(message);
});

resetBtn.addEventListener('click', async () => {
  await fetch('/api/reset', {method: 'POST'});
  append('assistant', '已重置对话。');
});
</script>
</body>
</html>`
