package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type userLine struct {
	Type    string  `json:"type"`
	Message userMsg `json:"message"`
}

type userMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type initLine struct {
	Type    string   `json:"type"`
	Subtype string   `json:"subtype"`
	Tools   []string `json:"tools"`
}

func main() {
	cases := []string{"", "Bash"}
	for i, tools := range cases {
		fmt.Printf("\n=== Case %d: --tools %q ===\n", i+1, tools)
		list, err := fetchToolsList(tools)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Printf("tools: %v\n", list)
	}
}

func fetchToolsList(tools string) ([]string, error) {
	args := []string{"--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--tools", tools}
	cmd := exec.Command("claude", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	enc := json.NewEncoder(stdin)
	if err := enc.Encode(userLine{Type: "user", Message: userMsg{Role: "user", Content: "hi"}}); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return nil, err
	}
	_ = stdin.Close()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	deadline := time.After(20 * time.Second)
	for {
		select {
		case <-deadline:
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, fmt.Errorf("timeout waiting for init line")
		default:
			if !scanner.Scan() {
				_ = cmd.Wait()
				return nil, fmt.Errorf("stream ended before init line")
			}
			var init initLine
			if err := json.Unmarshal(scanner.Bytes(), &init); err != nil {
				continue
			}
			if init.Type == "system" && init.Subtype == "init" {
				_ = cmd.Wait()
				return init.Tools, nil
			}
		}
	}
}
