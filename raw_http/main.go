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

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	sess, err := NewSession(*endpoint, *model, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sess.SetSystemPrompt(*system)
	sess.SetTemperature(*temp)
	sess.SetMaxSteps(*maxSteps)

	if *react {
		tools, handlers := DefaultTools()
		sess.EnableTools(tools, handlers)
	}

	out, err := sess.Chat(ctx, *prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(out)
}
