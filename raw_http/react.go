package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

func DefaultTools() ([]Tool, map[string]ToolHandler) {
	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "now",
				Description: "Get current time.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"timezone": map[string]interface{}{
							"type":        "string",
							"description": "IANA timezone, e.g. Asia/Shanghai. Empty means local.",
						},
						"format": map[string]interface{}{
							"type":        "string",
							"description": "Go time layout, default RFC3339.",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "calculator",
				Description: "Evaluate a simple arithmetic expression with + - * / and parentheses.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "Arithmetic expression, e.g. (1+2)*3/4",
						},
					},
					"required": []string{"expression"},
				},
			},
		},
	}

	handlers := map[string]ToolHandler{
		"now": func(ctx context.Context, args json.RawMessage) (string, error) {
			var in struct {
				Timezone string `json:"timezone"`
				Format   string `json:"format"`
			}
			if len(args) > 0 && string(args) != "null" {
				if err := json.Unmarshal(args, &in); err != nil {
					return "", fmt.Errorf("invalid args: %w", err)
				}
			}

			loc := time.Local
			if tz := strings.TrimSpace(in.Timezone); tz != "" {
				l, err := time.LoadLocation(tz)
				if err != nil {
					return "", fmt.Errorf("invalid timezone: %w", err)
				}
				loc = l
			}

			layout := time.RFC3339
			if f := strings.TrimSpace(in.Format); f != "" {
				layout = f
			}
			return time.Now().In(loc).Format(layout), nil
		},
		"calculator": func(ctx context.Context, args json.RawMessage) (string, error) {
			var in struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal(args, &in); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}
			in.Expression = strings.TrimSpace(in.Expression)
			if in.Expression == "" {
				return "", errors.New("empty expression")
			}
			v, err := evalArithmetic(in.Expression)
			if err != nil {
				return "", err
			}
			return strconv.FormatFloat(v, 'g', -1, 64), nil
		},
	}
	return tools, handlers
}

func doReACT(ctx context.Context, client *Client, model, systemPrompt, userPrompt string, tools []Tool, handlers map[string]ToolHandler, temperature float64, maxSteps int) (string, error) {
	if maxSteps <= 0 {
		return "", errors.New("maxSteps must be > 0")
	}
	if len(tools) > 0 && handlers == nil {
		return "", errors.New("handlers is nil")
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	for step := 0; step < maxSteps; step++ {
		resp, err := client.Invoke(ctx, ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: temperature,
			Stream:      false,
			Tools:       tools,
			ToolChoice:  "auto",
		})
		if err != nil {
			return "", err
		}
		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
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

	return "", fmt.Errorf("exceeded max steps (%d)", maxSteps)
}

type arithTokenKind int

const (
	arithNumber arithTokenKind = iota
	arithOp
	arithLParen
	arithRParen
)

type arithToken struct {
	kind  arithTokenKind
	num   float64
	op    string
	unary bool
}

func evalArithmetic(expr string) (float64, error) {
	tokens, err := tokenizeArithmetic(expr)
	if err != nil {
		return 0, err
	}
	rpn, err := toRPN(tokens)
	if err != nil {
		return 0, err
	}
	return evalRPN(rpn)
}

func tokenizeArithmetic(s string) ([]arithToken, error) {
	var out []arithToken
	for i := 0; i < len(s); {
		r := rune(s[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}

		switch s[i] {
		case '+', '-', '*', '/':
			out = append(out, arithToken{kind: arithOp, op: string(s[i])})
			i++
			continue
		case '(':
			out = append(out, arithToken{kind: arithLParen})
			i++
			continue
		case ')':
			out = append(out, arithToken{kind: arithRParen})
			i++
			continue
		}

		if (s[i] >= '0' && s[i] <= '9') || s[i] == '.' {
			start := i
			dotCount := 0
			for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
				if s[i] == '.' {
					dotCount++
					if dotCount > 1 {
						return nil, fmt.Errorf("invalid number near %q", s[start:i+1])
					}
				}
				i++
			}
			v, err := strconv.ParseFloat(s[start:i], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q: %w", s[start:i], err)
			}
			out = append(out, arithToken{kind: arithNumber, num: v})
			continue
		}

		return nil, fmt.Errorf("unexpected character %q", s[i])
	}
	return out, nil
}

func toRPN(tokens []arithToken) ([]arithToken, error) {
	var output []arithToken
	var ops []arithToken

	prevCanBeUnary := true
	for _, tok := range tokens {
		switch tok.kind {
		case arithNumber:
			output = append(output, tok)
			prevCanBeUnary = false
		case arithLParen:
			ops = append(ops, tok)
			prevCanBeUnary = true
		case arithRParen:
			found := false
			for len(ops) > 0 {
				top := ops[len(ops)-1]
				ops = ops[:len(ops)-1]
				if top.kind == arithLParen {
					found = true
					break
				}
				output = append(output, top)
			}
			if !found {
				return nil, errors.New("mismatched parentheses")
			}
			prevCanBeUnary = false
		case arithOp:
			if prevCanBeUnary && (tok.op == "+" || tok.op == "-") {
				tok.unary = true
				tok.op = "u" + tok.op
			}

			for len(ops) > 0 {
				top := ops[len(ops)-1]
				if top.kind != arithOp {
					break
				}
				if (isRightAssociative(tok) && precedence(top) > precedence(tok)) || (!isRightAssociative(tok) && precedence(top) >= precedence(tok)) {
					ops = ops[:len(ops)-1]
					output = append(output, top)
					continue
				}
				break
			}
			ops = append(ops, tok)
			prevCanBeUnary = true
		default:
			return nil, errors.New("unknown token")
		}
	}

	for len(ops) > 0 {
		top := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		if top.kind == arithLParen || top.kind == arithRParen {
			return nil, errors.New("mismatched parentheses")
		}
		output = append(output, top)
	}
	return output, nil
}

func precedence(tok arithToken) int {
	switch tok.op {
	case "u+", "u-":
		return 3
	case "*", "/":
		return 2
	case "+", "-":
		return 1
	default:
		return 0
	}
}

func isRightAssociative(tok arithToken) bool {
	return tok.op == "u+" || tok.op == "u-"
}

func evalRPN(tokens []arithToken) (float64, error) {
	var stack []float64
	for _, tok := range tokens {
		switch tok.kind {
		case arithNumber:
			stack = append(stack, tok.num)
		case arithOp:
			if tok.op == "u+" || tok.op == "u-" {
				if len(stack) < 1 {
					return 0, errors.New("invalid expression")
				}
				v := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if tok.op == "u-" {
					v = -v
				}
				stack = append(stack, v)
				continue
			}

			if len(stack) < 2 {
				return 0, errors.New("invalid expression")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var v float64
			switch tok.op {
			case "+":
				v = a + b
			case "-":
				v = a - b
			case "*":
				v = a * b
			case "/":
				if b == 0 {
					return 0, errors.New("division by zero")
				}
				v = a / b
			default:
				return 0, fmt.Errorf("unknown operator %q", tok.op)
			}
			if math.IsInf(v, 0) || math.IsNaN(v) {
				return 0, errors.New("invalid numeric result")
			}
			stack = append(stack, v)
		default:
			return 0, errors.New("invalid token in rpn")
		}
	}
	if len(stack) != 1 {
		return 0, errors.New("invalid expression")
	}
	return stack[0], nil
}
