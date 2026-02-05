package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("claude", "--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	_ = cmd.Start()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	fmt.Printf("\n--- Round 1 ---\n")
	b1, _ := json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": "你好，请记住一个词：苹果。",
		},
	})
	_, _ = stdin.Write(append(b1, '\n'))
	for scanner.Scan() {
		var line map[string]any
		_ = json.Unmarshal(scanner.Bytes(), &line)
		if line["type"] == "assistant" {
			fmt.Println(string(scanner.Bytes()))
			break
		}
	}

	fmt.Printf("\n--- Round 2 ---\n")
	b2, _ := json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": "刚才让你记住的词是什么？只回答这个词。",
		},
	})
	_, _ = stdin.Write(append(b2, '\n'))
	for scanner.Scan() {
		var line map[string]any
		_ = json.Unmarshal(scanner.Bytes(), &line)
		if line["type"] == "assistant" {
			fmt.Println(string(scanner.Bytes()))
			break
		}
	}

	_ = stdin.Close()
	_ = cmd.Wait()
}
