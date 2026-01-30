package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

type Client struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(endpoint, apiKey string, timeout time.Duration) *Client {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		Endpoint: strings.TrimSpace(endpoint),
		APIKey:   strings.TrimSpace(apiKey),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`

	Choices []struct {
		Index        int     `json:"index,omitempty"`
		Message      Message `json:"message,omitempty"`
		FinishReason string  `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`

	Usage *struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`

	Error *struct {
		Message string      `json:"message,omitempty"`
		Type    string      `json:"type,omitempty"`
		Code    interface{} `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

type InvokeResult struct {
	Endpoint   string
	StatusCode int
	Duration   time.Duration

	Request     ChatCompletionRequest
	RawRequest  []byte
	RawResponse []byte
	Response    ChatCompletionResponse
}

func (c *Client) Invoke(ctx context.Context, req ChatCompletionRequest) (*InvokeResult, error) {
	if strings.TrimSpace(c.Endpoint) == "" {
		return nil, errors.New("empty endpoint")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New("empty api key")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	out := &InvokeResult{
		Endpoint:    c.Endpoint,
		StatusCode:  resp.StatusCode,
		Duration:    time.Since(start),
		Request:     req,
		RawRequest:  payload,
		RawResponse: bodyBytes,
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(bodyBytes))
		if msg == "" {
			msg = resp.Status
		}
		return out, fmt.Errorf("http %d: %s", resp.StatusCode, msg)
	}

	if err := json.Unmarshal(bodyBytes, &out.Response); err != nil {
		return out, fmt.Errorf("decode json: %w", err)
	}
	if out.Response.Error != nil && strings.TrimSpace(out.Response.Error.Message) != "" {
		return out, fmt.Errorf("api error: %s", strings.TrimSpace(out.Response.Error.Message))
	}
	if len(out.Response.Choices) == 0 {
		return out, errors.New("empty choices")
	}
	return out, nil
}
