package main

import "strings"

const ThinkInstructions = `请在回答时输出思考过程与最终答案，格式如下：
<think>这里写思考过程</think>
<final>这里写最终答案</final>`

func WithThink(systemPrompt string) string {
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt == "" {
		return ThinkInstructions
	}
	return systemPrompt + "\n\n" + ThinkInstructions
}
