package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func RenderInvokeResult(invoke *InvokeResult) string {
	if invoke == nil {
		return ""
	}
	msgs := cloneMessages(invoke.Request.Messages)
	if len(invoke.Response.Choices) > 0 {
		msgs = append(msgs, invoke.Response.Choices[0].Message)
	}

	res := &ReACTResult{
		BaseMessagesLen: len(invoke.Request.Messages),
		Messages:        msgs,
		Invokes:         []*InvokeResult{invoke},
	}

	var b strings.Builder
	fmt.Fprintf(&b, "== Invoke (status=%d, latency=%s) ==\n\n", invoke.StatusCode, invoke.Duration.Round(time.Millisecond))
	b.WriteString(RenderReACTResult(res))
	return b.String()
}

func RenderReACTResult(res *ReACTResult) string {
	if res == nil {
		return ""
	}
	var b strings.Builder

	invokeIdx := 0
	for i, m := range res.Messages {
		label := m.Role
		switch m.Role {
		case "assistant":
			if i >= res.BaseMessagesLen && invokeIdx < len(res.Invokes) {
				inv := res.Invokes[invokeIdx]
				invokeIdx++
				if inv != nil && len(inv.Response.Choices) > 0 {
					finish := strings.TrimSpace(inv.Response.Choices[0].FinishReason)
					if finish == "" {
						finish = "unknown"
					}
					label = fmt.Sprintf("assistant#%d (finish=%s, latency=%s)", invokeIdx, finish, inv.Duration.Round(time.Millisecond))
					if inv.Response.Usage != nil && inv.Response.Usage.TotalTokens > 0 {
						label = fmt.Sprintf("%s, tokens=%d", label, inv.Response.Usage.TotalTokens)
					}
				} else {
					label = fmt.Sprintf("assistant#%d", invokeIdx)
				}
			}
		case "tool":
			if strings.TrimSpace(m.ToolCallID) != "" {
				label = fmt.Sprintf("tool[%s]", m.ToolCallID)
			} else {
				label = "tool"
			}
		}

		fmt.Fprintf(&b, "%02d %s:\n", i+1, label)

		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			b.WriteString("  tool_calls:\n")
			for _, tc := range m.ToolCalls {
				fmt.Fprintf(&b, "  - %s (id=%s)\n", tc.Function.Name, tc.ID)
				args := prettyMaybeJSON(tc.Function.Arguments)
				if args != "" {
					b.WriteString("    arguments:\n")
					b.WriteString(indentBlock(args, "      "))
					b.WriteByte('\n')
				}
			}
		}

		content := strings.TrimSpace(m.Content)
		if content != "" {
			if m.Role == "tool" {
				content = prettyMaybeJSON(content)
			}
			b.WriteString(indentBlock(content, "  "))
			b.WriteByte('\n')
		}

		b.WriteByte('\n')
	}
	return b.String()
}

func indentBlock(s, prefix string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func prettyMaybeJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	var out bytes.Buffer
	out.Write(b)
	return out.String()
}
