package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSession_MultiTurn_WithTools_Integration(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("QWEN_API_KEY"))
	if apiKey == "" {
		t.Skip("set QWEN_API_KEY to run integration test")
	}

	sess, err := NewSession(DefaultEndpoint, "qwen-plus", 60*time.Second)
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	sess.SetSystemPrompt("You are a helpful assistant. Use tools when asked.")
	sess.SetTemperature(0)

	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_current_date",
				Description: "Get today's date.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
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

	handlers := map[string]ToolHandler{
		"get_current_date": func(ctx context.Context, _ json.RawMessage) (string, error) {
			return time.Now().Format("2006-01-02"), nil
		},
		"get_current_weather": func(ctx context.Context, args json.RawMessage) (string, error) {
			var in struct {
				Location string `json:"location"`
				Unit     string `json:"unit"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			if strings.TrimSpace(in.Unit) == "" {
				in.Unit = "celsius"
			}
			return `{"weather":"sunny","temperature":20,"unit":"` + in.Unit + `"}`, nil
		},
	}
	sess.EnableTools(tools, handlers)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	turns := []string{
		"你好，用一句话介绍你自己。",
		"今天是几号？请调用 get_current_date 工具。",
		"上海今天天气如何？请调用 get_current_weather 工具并使用 celsius。",
	}

	for i, prompt := range turns {
		if _, err := sess.Chat(ctx, prompt); err != nil {
			t.Fatalf("turn%d error: %v", i+1, err)
		}
	}

	if got := len(sess.Messages()); got < 2*len(turns)+1 {
		t.Fatalf("history len = %d, want at least %d", got, 2*len(turns)+1)
	}
}
