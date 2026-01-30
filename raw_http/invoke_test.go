package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientInvoke_HappyPath(t *testing.T) {
	apiKey := "test-api-key"

	var gotReq ChatCompletionRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Errorf("Authorization = %q, want %q", got, "Bearer "+apiKey)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Errorf("decode request json: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, apiKey, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Invoke(ctx, ChatCompletionRequest{
		Model:    "qwen-plus",
		Messages: []Message{{Role: "system", Content: "s"}, {Role: "user", Content: "u"}},
		Stream:   false,
	})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if resp.Choices[0].Message.Content != "ok" {
		t.Fatalf("content = %q, want %q", resp.Choices[0].Message.Content, "ok")
	}

	if gotReq.Model != "qwen-plus" {
		t.Fatalf("sent model = %q, want %q", gotReq.Model, "qwen-plus")
	}
	if len(gotReq.Messages) != 2 || gotReq.Messages[1].Role != "user" || gotReq.Messages[1].Content != "u" {
		t.Fatalf("sent messages = %#v, want system+user", gotReq.Messages)
	}
}
