package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSession_MultiTurn_Integration_HappyPath(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("QWEN_API_KEY"))
	if apiKey == "" {
		t.Skip("set QWEN_API_KEY to run integration test")
	}

	sess, err := NewSession(DefaultEndpoint, "qwen-plus", 60*time.Second)
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	sess.SetSystemPrompt("You are a helpful assistant.")
	sess.SetTemperature(0)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := sess.Chat(ctx, "Reply with one short sentence: hello."); err != nil {
		t.Fatalf("turn1 error: %v", err)
	}
	if got := len(sess.Messages()); got != 3 {
		t.Fatalf("history len after turn1 = %d, want %d", got, 3)
	}

	if _, err := sess.Chat(ctx, "Reply with one short sentence: ok."); err != nil {
		t.Fatalf("turn2 error: %v", err)
	}
	if got := len(sess.Messages()); got != 5 {
		t.Fatalf("history len after turn2 = %d, want %d", got, 5)
	}
}
