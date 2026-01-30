package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Session struct {
	client      *Client
	model       string
	temperature float64
	maxSteps    int

	tools    []Tool
	handlers map[string]ToolHandler

	messages []Message
}

func NewSession(endpoint, model string, timeout time.Duration) (*Session, error) {
	apiKey := strings.TrimSpace(os.Getenv("QWEN_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("missing API key: set QWEN_API_KEY")
	}
	if strings.TrimSpace(model) == "" {
		model = "qwen-plus"
	}
	return &Session{
		client:      NewClient(endpoint, apiKey, timeout),
		model:       model,
		temperature: 0.7,
		maxSteps:    8,
	}, nil
}

func (s *Session) SetSystemPrompt(prompt string) {
	prompt = strings.TrimSpace(prompt)
	if len(s.messages) > 0 && s.messages[0].Role == "system" {
		s.messages[0].Content = prompt
		return
	}
	s.messages = append([]Message{{Role: "system", Content: prompt}}, s.messages...)
}

func (s *Session) SetTemperature(t float64) { s.temperature = t }

func (s *Session) EnableTools(tools []Tool, handlers map[string]ToolHandler) {
	s.tools = tools
	s.handlers = handlers
}

func (s *Session) SetMaxSteps(n int) { s.maxSteps = n }

func (s *Session) Messages() []Message {
	out := make([]Message, len(s.messages))
	copy(out, s.messages)
	return out
}

func (s *Session) Reset(keepSystem bool) {
	if !keepSystem {
		s.messages = nil
		return
	}
	if len(s.messages) > 0 && s.messages[0].Role == "system" {
		s.messages = s.messages[:1]
		return
	}
	s.messages = nil
}

func (s *Session) Chat(ctx context.Context, userPrompt string) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("nil session")
	}
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return "", errors.New("empty user prompt")
	}

	origLen := len(s.messages)
	s.messages = append(s.messages, Message{Role: "user", Content: userPrompt})

	if len(s.tools) > 0 {
		updated, out, err := doReACTWithHistory(ctx, s.client, s.model, s.messages, s.tools, s.handlers, s.temperature, s.maxSteps)
		if err != nil {
			s.messages = s.messages[:origLen]
			return "", err
		}
		s.messages = updated
		return out, nil
	}

	resp, err := s.client.Invoke(ctx, ChatCompletionRequest{
		Model:       s.model,
		Messages:    s.messages,
		Temperature: s.temperature,
		Stream:      false,
	})
	if err != nil {
		s.messages = s.messages[:origLen]
		return "", err
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) > 0 {
		s.messages = s.messages[:origLen]
		return "", fmt.Errorf("model returned tool_calls; enable tools to execute them")
	}

	s.messages = append(s.messages, msg)
	return msg.Content, nil
}

