package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	args := normalizeArgs(os.Args[1:])
	cmd := exec.Command("claude", args...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start claude: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdinFd) {
		return fmt.Errorf("stdin is not a terminal")
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return fmt.Errorf("make raw: %w", err)
	}
	defer func() { _ = term.Restore(stdinFd, oldState) }()

	resize := func() {
		if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
			fmt.Fprintln(os.Stderr, "warn: resize:", err)
		}
	}
	resize()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			resize()
		}
	}()
	defer signal.Stop(sigCh)

	outputErr := make(chan error, 1)
	go func() { _, err := io.Copy(ptmx, os.Stdin); _ = err }()
	go func() { _, err := io.Copy(os.Stdout, ptmx); outputErr <- err }()

	err = cmd.Wait()
	_ = ptmx.Close() // unblock io.Copy on stdin when child exits
	<-outputErr

	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return fmt.Errorf("claude exited: %d", status.ExitStatus())
		}
	}
	return err
}

func normalizeArgs(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		// Allow "go run . -- <args>" even though "go run" doesn't treat "--" specially.
		return args[1:]
	}
	return args
}
