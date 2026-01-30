package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		endpoint = flag.String("endpoint", DefaultEndpoint, "DashScope OpenAI-compatible endpoint")
		model    = flag.String("model", "qwen-plus", "model name, e.g. qwen-plus / qwen-turbo")
		system   = flag.String("system", "You are a helpful assistant.", "system prompt")
		prompt   = flag.String("prompt", "", "user prompt")
		temp     = flag.Float64("temp", 0.7, "temperature")
		timeout  = flag.Duration("timeout", 60*time.Second, "request timeout")
		react    = flag.Bool("react", false, "enable ReACT loop with tool-calls")
		maxSteps = flag.Int("max-steps", 8, "ReACT max steps")
	)
	flag.Parse()

	if strings.TrimSpace(*prompt) == "" {
		fmt.Fprintln(os.Stderr, "missing -prompt")
		os.Exit(2)
	}

	apiKey := strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("ALIYUN_DASHSCOPE_API_KEY"))
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing API key: set DASHSCOPE_API_KEY (or ALIYUN_DASHSCOPE_API_KEY)")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := NewClient(*endpoint, apiKey, *timeout)

	if *react {
		tools, handlers := DefaultTools()
		out, err := doReACT(ctx, client, *model, *system, *prompt, tools, handlers, *temp, *maxSteps)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(out)
		return
	}

	resp, err := client.Invoke(ctx, ChatCompletionRequest{
		Model:       *model,
		Messages:    []Message{{Role: "system", Content: *system}, {Role: "user", Content: *prompt}},
		Temperature: *temp,
		Stream:      false,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) > 0 {
		fmt.Fprintln(os.Stderr, "model returned tool_calls; rerun with -react to execute tools")
		os.Exit(1)
	}
	fmt.Print(msg.Content)
}
