package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

type ReACTResult struct {
	Final           string
	BaseMessagesLen int
	Messages        []Message
	Invokes         []*InvokeResult
}

func doReACTWithHistory(ctx context.Context, client *Client, model string, messages []Message, tools []Tool, handlers map[string]ToolHandler, temperature float64, maxSteps int) (*ReACTResult, error) {
	history := cloneMessages(messages)
	result := &ReACTResult{
		BaseMessagesLen: len(history),
		Messages:        history,
	}

	if maxSteps <= 0 {
		return result, errors.New("maxSteps must be > 0")
	}
	if len(tools) > 0 && handlers == nil {
		return result, errors.New("handlers is nil")
	}

	for step := 0; step < maxSteps; step++ {
		req := ChatCompletionRequest{
			Model:       model,
			Messages:    history,
			Temperature: temperature,
			Stream:      false,
			Tools:       tools,
		}
		if len(tools) > 0 {
			req.ToolChoice = "auto"
		}

		invoke, err := client.Invoke(ctx, req)
		result.Invokes = append(result.Invokes, invoke)
		if err != nil {
			result.Messages = history
			return result, err
		}

		msg := invoke.Response.Choices[0].Message
		history = append(history, msg)

		if len(msg.ToolCalls) == 0 {
			result.Final = msg.Content
			result.Messages = history
			return result, nil
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
			history = append(history, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
	}

	result.Messages = history
	return result, fmt.Errorf("exceeded max steps (%d)", maxSteps)
}

func doReACT(ctx context.Context, client *Client, model, systemPrompt, userPrompt string, tools []Tool, handlers map[string]ToolHandler, temperature float64, maxSteps int) (*ReACTResult, error) {
	return doReACTWithHistory(ctx, client, model, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, tools, handlers, temperature, maxSteps)
}
