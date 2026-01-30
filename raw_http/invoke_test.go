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

	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_current_weather",
				Description: "Get the current weather in a given location.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City and state, e.g. San Francisco, CA",
						},
						"unit": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"celsius", "fahrenheit"},
							"description": "Temperature unit.",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	resp, err := client.Invoke(ctx, ChatCompletionRequest{
		Model: "qwen-plus",
		Messages: []Message{
			{Role: "system", Content: "You are a tool-calling assistant. You must call the provided tool to answer. Do not answer directly."},
			{Role: "user", Content: "What's the current weather in San Francisco? Call get_current_weather with unit=celsius."},
		},
		Temperature: 0,
		Stream:      false,
		Tools:       tools,
		ToolChoice:  "required",
	})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		t.Fatalf("expected tool_calls, got none (content=%q)", msg.Content)
	}
	t.Log(msg)
}
