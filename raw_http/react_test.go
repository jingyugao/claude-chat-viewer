package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoReACT_HappyPath_ToolCall(t *testing.T) {
	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name: "echo",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{"type": "string"},
					},
					"required": []string{"text"},
				},
			},
		},
	}
	handlers := map[string]ToolHandler{
		"echo": func(ctx context.Context, args json.RawMessage) (string, error) {
			var in struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			return "ECHO:" + in.Text, nil
		},
	}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := calls.Add(1)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var req ChatCompletionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch step {
		case 1:
			_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"echo","arguments":"{\"text\":\"abc\"}"}}]},"finish_reason":"tool_calls"}]}`)
		case 2:
			if len(req.Messages) < 1 || req.Messages[len(req.Messages)-1].Role != "tool" || req.Messages[len(req.Messages)-1].ToolCallID != "call_1" || req.Messages[len(req.Messages)-1].Content != "ECHO:abc" {
				http.Error(w, "missing tool result in messages", http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"final"},"finish_reason":"stop"}]}`)
		default:
			http.Error(w, "too many calls", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "k", 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := doReACT(ctx, client, "qwen-plus", "s", "u", tools, handlers, 0.1, 4)
	if err != nil {
		t.Fatalf("doReACT error: %v", err)
	}
	if out != "final" {
		t.Fatalf("out = %q, want %q", out, "final")
	}
}

func TestEvalArithmetic_HappyPath(t *testing.T) {
	got, err := evalArithmetic("(1+2)*3/4")
	if err != nil {
		t.Fatalf("evalArithmetic error: %v", err)
	}
	if got != 2.25 {
		t.Fatalf("got = %v, want %v", got, 2.25)
	}
}

func TestDefaultTools_HappyPath(t *testing.T) {
	_, handlers := DefaultTools()
	calc := handlers["calculator"]
	now := handlers["now"]

	out, err := calc(context.Background(), json.RawMessage(`{"expression":"(1+2)*3"}`))
	if err != nil {
		t.Fatalf("calculator error: %v", err)
	}
	if out != "9" {
		t.Fatalf("calculator out = %q, want %q", out, "9")
	}

	s, err := now(context.Background(), json.RawMessage(`{"timezone":"UTC","format":"2006-01-02T15:04:05Z07:00"}`))
	if err != nil {
		t.Fatalf("now error: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		t.Fatalf("now output parse error: %v (value=%q)", err, s)
	}
}
