# Claude Pty Demo (Go)

This demo launches `claude` as a child process with a pseudo‑terminal so you can interact with it directly (like the real CLI).

## Prereqs
- `claude` CLI is installed and available in `PATH`
- You are already logged in to Claude
- Go 1.22+

## Run
```bash
go run .
```

Pass any arguments after `--` and they will be forwarded to `claude`:
```bash
go run . -p "1+1=? (Answer in 1 word)"
```

If you accidentally include `--`, the demo will ignore it:
```bash
go run . -- -p "1+1=? (Answer in 1 word)"
```

## Multi-turn in a Single Process (stream-json)
This demo keeps one Claude process alive and streams multiple user turns through stdin.
It uses the supported `--print --input-format stream-json` interface.

```bash
go run ./cmd/stream
```

## System Prompt Override
This demo verifies `--system-prompt` can change the system instructions.

```bash
go run ./cmd/system_prompt
```

## Tools Restriction
This demo checks that `--tools` controls the allowed tool list by reading the init line from `stream-json`.

```bash
go run ./cmd/tools
```

## K8s Services Analysis (kubectl)
This demo shells out to `kubectl get svc -o json`, then summarizes services by type and flags common issues.

```bash
go run ./cmd/k8s_services -context my-context -namespace default
```

## K8s Agent (Claude + kubectl)
This demo wraps a K8s agent in Go. It gathers `kubectl` service data, summarizes it, then asks Claude to answer.

```bash
go run ./cmd/k8s_agent -q "哪些 LoadBalancer 还没有外网 IP？"
```

Use Claude's Bash tool directly (no local kubectl parsing):
```bash
go run ./cmd/k8s_agent -mode bash -q "哪些服务负载比较高？"
```

## K8s Agent Web UI
This demo starts a web server with a chat UI for multi-turn questions about the cluster.

```bash
go run ./cmd/k8s_web -addr :8080
```

Run web UI in bash mode:
```bash
go run ./cmd/k8s_web -addr :8080 -mode bash
```

## Notes
- The program puts your terminal into raw mode to support full interactive behavior.
- Window resize is handled via `SIGWINCH` and propagated to the pty.
