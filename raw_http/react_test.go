package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDoReACT_Integration_ThreeToolChain(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("QWEN_API_KEY"))
	if apiKey == "" {
		t.Skip("set QWEN_API_KEY to run integration test")
	}

	var (
		callOrder   []string
		dateOut     = time.Now().Format("2006-01-02")
		locationOut = "San Francisco, CA"
		weatherArgs struct{ Date, Location, Unit string }
	)

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
				Name:        "get_current_location",
				Description: "Get user's current location (city, country).",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_weather_by_date",
				Description: "Get weather by date and location.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"date": map[string]interface{}{
							"type":        "string",
							"description": "Date in YYYY-MM-DD",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City and region, e.g. San Francisco, CA",
						},
						"unit": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"celsius", "fahrenheit"},
							"description": "Temperature unit.",
						},
					},
					"required": []string{"date", "location"},
				},
			},
		},
	}

	handlers := map[string]ToolHandler{
		"get_current_date": func(ctx context.Context, args json.RawMessage) (string, error) {
			callOrder = append(callOrder, "get_current_date")
			return dateOut, nil
		},
		"get_current_location": func(ctx context.Context, args json.RawMessage) (string, error) {
			callOrder = append(callOrder, "get_current_location")
			return locationOut, nil
		},
		"get_weather_by_date": func(ctx context.Context, args json.RawMessage) (string, error) {
			callOrder = append(callOrder, "get_weather_by_date")
			var in struct {
				Date     string `json:"date"`
				Location string `json:"location"`
				Unit     string `json:"unit"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", err
			}
			weatherArgs.Date = in.Date
			weatherArgs.Location = in.Location
			weatherArgs.Unit = in.Unit
			return `{"weather":"sunny","temperature":20,"unit":"celsius"}`, nil
		},
	}

	system := `You are a tool-calling assistant.
Follow this exact plan and call exactly ONE tool per step:
1) Call get_current_date with {}.
2) Call get_current_location with {}.
3) Call get_weather_by_date with {"date":<date>,"location":<location>,"unit":"celsius"} using outputs from previous tools.
Do not answer until step 3 is completed. Then answer in Chinese in 1 sentence.`

	client := NewClient(DefaultEndpoint, apiKey, 60*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	res, err := doReACT(ctx, client, "qwen-plus", system, "今天天气如何？", tools, handlers, 0, 12)
	if err != nil {
		t.Fatalf("doReACT error: %v", err)
	}
	if len(callOrder) < 3 {
		t.Fatalf("callOrder = %#v, want at least 3 tool calls", callOrder)
	}
	if callOrder[0] != "get_current_date" || callOrder[1] != "get_current_location" || callOrder[2] != "get_weather_by_date" {
		t.Fatalf("callOrder first 3 = %#v, want [get_current_date get_current_location get_weather_by_date]", callOrder[:3])
	}
	if strings.TrimSpace(weatherArgs.Date) == "" || strings.TrimSpace(weatherArgs.Location) == "" {
		t.Fatalf("weather tool args missing fields: %+v", weatherArgs)
	}
	if strings.TrimSpace(res.Final) == "" {
		t.Fatalf("empty final answer")
	}
	if len(res.Invokes) == 0 {
		t.Fatalf("expected invokes trace, got none")
	}
	if len(res.Messages) <= res.BaseMessagesLen {
		t.Fatalf("expected messages to grow (base=%d, got=%d)", res.BaseMessagesLen, len(res.Messages))
	}
	t.Log(res.Final)
}
