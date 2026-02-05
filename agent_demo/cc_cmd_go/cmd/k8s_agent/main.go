package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gao/claude-chat-viewer/agent_demo/cc_cmd_go/pkg/k8sagent"
)

func main() {
	var kubeconfig string
	var kubeContext string
	var namespace string
	var timeout time.Duration
	var model string
	var tools string
	var systemPrompt string
	var mode string
	var question string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	flag.StringVar(&kubeContext, "context", "", "Kube context")
	flag.StringVar(&namespace, "namespace", "", "Namespace filter (default all)")
	flag.DurationVar(&timeout, "timeout", 20*time.Second, "kubectl timeout")
	flag.StringVar(&model, "model", "", "Claude model name (optional)")
	flag.StringVar(&tools, "tools", "", "Claude --tools value (optional)")
	flag.StringVar(&systemPrompt, "system", "", "Extra system prompt to append")
	flag.StringVar(&mode, "mode", "summary", "Agent mode: summary or bash")
	flag.StringVar(&question, "q", "请总结集群 Service 的类型分布与异常项。", "Question to ask")
	flag.Parse()

	agent := k8sagent.New(k8sagent.Config{
		Kubeconfig:   kubeconfig,
		Context:      kubeContext,
		Namespace:    namespace,
		Timeout:      timeout,
		Model:        model,
		Tools:        tools,
		SystemPrompt: systemPrompt,
		Mode:         k8sagent.Mode(mode),
	})

	ctx := context.Background()
	answer, err := agent.Ask(ctx, question)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
