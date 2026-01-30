package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClientInvoke_Integration_HappyPath(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("QWEN_API_KEY"))
	if apiKey == "" {
		t.Skip("set QWEN_API_KEY to run integration test")
	}

	client := NewClient(DefaultEndpoint, apiKey, 60*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Invoke(ctx, ChatCompletionRequest{
		Model:       "qwen-plus",
		Messages:    []Message{{Role: "system", Content: "You are a helpful assistant."}, {Role: "user", Content: "Reply with a single word: ok"}},
		Temperature: 0,
		Stream:      false,
	})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		t.Fatalf("empty assistant content")
	}
}
