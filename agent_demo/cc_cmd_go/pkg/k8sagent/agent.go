package k8sagent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	KubectlPath string
	Kubeconfig  string
	Context     string
	Namespace   string
	Timeout     time.Duration

	Model          string
	SystemPrompt   string
	Tools          string
	PermissionMode string
	Mode           Mode
}

type ChatMessage struct {
	Role    string
	Content string
}

type Mode string

const (
	ModeSummary Mode = "summary"
	ModeBash    Mode = "bash"
)

type Agent struct {
	cfg Config
}

func New(cfg Config) *Agent {
	if cfg.Timeout == 0 {
		cfg.Timeout = 20 * time.Second
	}
	return &Agent{cfg: cfg}
}

func (a *Agent) Ask(ctx context.Context, question string) (string, error) {
	return a.AskWithHistory(ctx, question, nil)
}

func (a *Agent) AskWithHistory(ctx context.Context, question string, history []ChatMessage) (string, error) {
	if a.cfg.Mode == ModeBash {
		return a.askViaBash(ctx, question, history)
	}
	list, err := FetchServices(ctx, a.cfg.KubectlPath, a.cfg.Kubeconfig, a.cfg.Context, a.cfg.Namespace, a.cfg.Timeout)
	if err != nil {
		return "", err
	}

	summary := RenderSummary(Summarize(list), a.cfg.Namespace)
	systemPrompt := a.systemPrompt()
	userPrompt := buildUserPromptWithHistory(question, summary, history)

	args := []string{"--print", "--system-prompt", systemPrompt, "-p", userPrompt}
	if a.cfg.Model != "" {
		args = append([]string{"--model", a.cfg.Model}, args...)
	}
	if a.cfg.Tools != "" {
		args = append(args, "--tools", a.cfg.Tools)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude error: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *Agent) askViaBash(ctx context.Context, question string, history []ChatMessage) (string, error) {
	tools := strings.TrimSpace(a.cfg.Tools)
	if tools == "" {
		tools = "Bash"
	}
	permission := strings.TrimSpace(a.cfg.PermissionMode)
	if permission == "" {
		permission = "bypassPermissions"
	}

	systemPrompt := a.bashSystemPrompt()
	userPrompt := buildBashPrompt(question, history, a.kubectlBaseHint())

	args := []string{
		"--print",
		"--tools", tools,
		"--permission-mode", permission,
		"--system-prompt", systemPrompt,
		"-p", userPrompt,
	}
	if a.cfg.Model != "" {
		args = append([]string{"--model", a.cfg.Model}, args...)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude error: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *Agent) systemPrompt() string {
	base := "You are a Kubernetes service analysis assistant. Use ONLY the provided service summary. If information is missing, say so. Always answer in Chinese."
	if strings.TrimSpace(a.cfg.SystemPrompt) == "" {
		return base
	}
	return base + "\n\n" + a.cfg.SystemPrompt
}

func (a *Agent) bashSystemPrompt() string {
	base := "You are a Kubernetes analysis assistant. You MUST use the Bash tool to run read-only kubectl commands (get/describe/top) to answer. Never mutate the cluster. If data is missing, say so. Always answer in Chinese."
	if strings.TrimSpace(a.cfg.SystemPrompt) == "" {
		return base
	}
	return base + "\n\n" + a.cfg.SystemPrompt
}

func buildUserPrompt(question, summary string) string {
	return buildUserPromptWithHistory(question, summary, nil)
}

func buildUserPromptWithHistory(question, summary string, history []ChatMessage) string {
	if strings.TrimSpace(summary) == "" {
		summary = "(no data)"
	}
	var b strings.Builder
	b.WriteString("Conversation history:\n")
	trimmed := trimHistory(history, 12)
	if len(trimmed) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, m := range trimmed {
			role := strings.TrimSpace(m.Role)
			if role == "" {
				role = "user"
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", role, strings.TrimSpace(m.Content)))
		}
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("User question: %s\n\nService summary:\n%s", question, summary))
	return b.String()
}

func trimHistory(history []ChatMessage, max int) []ChatMessage {
	if max <= 0 || len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}

func buildBashPrompt(question string, history []ChatMessage, kubectlBase string) string {
	var b strings.Builder
	b.WriteString("Conversation history:\n")
	trimmed := trimHistory(history, 12)
	if len(trimmed) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, m := range trimmed {
			role := strings.TrimSpace(m.Role)
			if role == "" {
				role = "user"
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", role, strings.TrimSpace(m.Content)))
		}
	}
	b.WriteString("\n")
	b.WriteString("Kubectl base command:\n")
	b.WriteString(kubectlBase)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("User question: %s", question))
	return b.String()
}

func (a *Agent) kubectlBaseHint() string {
	parts := []string{"kubectl"}
	if a.cfg.Kubeconfig != "" {
		parts = append(parts, "--kubeconfig", shellQuote(a.cfg.Kubeconfig))
	}
	if a.cfg.Context != "" {
		parts = append(parts, "--context", shellQuote(a.cfg.Context))
	}
	if a.cfg.Namespace != "" {
		parts = append(parts, "-n", shellQuote(a.cfg.Namespace))
	} else {
		parts = append(parts, "-A")
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
