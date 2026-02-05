package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	systemPrompt := "You are a strict validator. Respond with only the word 'YES'."
	userPrompt := "Please say hello."

	cmd := exec.Command(
		"claude",
		"--print",
		"--system-prompt", systemPrompt,
		"-p", userPrompt,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(1)
	}

	resp := strings.TrimSpace(string(out))
	fmt.Println("system prompt:", systemPrompt)
	fmt.Println("user prompt:", userPrompt)
	fmt.Println("assistant:", resp)
}
