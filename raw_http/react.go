package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

func doReACTWithHistory(ctx context.Context, client *Client, model string, messages []Message, tools []Tool, handlers map[string]ToolHandler, temperature float64, maxSteps int) ([]Message, string, error) {
	if maxSteps <= 0 {
		return nil, "", errors.New("maxSteps must be > 0")
	}
	if len(tools) > 0 && handlers == nil {
		return nil, "", errors.New("handlers is nil")
	}

	for step := 0; step < maxSteps; step++ {
		req := ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: temperature,
			Stream:      false,
			Tools:       tools,
		}
		if len(tools) > 0 {
			req.ToolChoice = "auto"
		}

		resp, err := client.Invoke(ctx, req)
		if err != nil {
			return nil, "", err
		}
		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) == 0 {
			return messages, msg.Content, nil
		}

		for _, tc := range msg.ToolCalls {
			handler, ok := handlers[tc.Function.Name]
			var out string
			if !ok {
				out = fmt.Sprintf("tool not found: %s", tc.Function.Name)
			} else {
				toolOut, err := handler(ctx, json.RawMessage(tc.Function.Arguments))
				if err != nil {
					out = fmt.Sprintf("tool error: %v", err)
				} else {
					out = toolOut
				}
			}
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
	}
	return nil, "", fmt.Errorf("exceeded max steps (%d)", maxSteps)
}

func doReACT(ctx context.Context, client *Client, model, systemPrompt, userPrompt string, tools []Tool, handlers map[string]ToolHandler, temperature float64, maxSteps int) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	_, out, err := doReACTWithHistory(ctx, client, model, messages, tools, handlers, temperature, maxSteps)
	return out, err
}
