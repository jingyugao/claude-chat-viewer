package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEvalArithmetic_HappyPath(t *testing.T) {
	got, err := evalArithmetic("(1+2)*3/4")
	if err != nil {
		t.Fatalf("evalArithmetic error: %v", err)
	}
	if got != 2.25 {
		t.Fatalf("got = %v, want %v", got, 2.25)
	}
}

func TestDefaultTools_HappyPath(t *testing.T) {
	_, handlers := DefaultTools()
	calc := handlers["calculator"]
	now := handlers["now"]

	out, err := calc(context.Background(), json.RawMessage(`{"expression":"(1+2)*3"}`))
	if err != nil {
		t.Fatalf("calculator error: %v", err)
	}
	if out != "9" {
		t.Fatalf("calculator out = %q, want %q", out, "9")
	}

	s, err := now(context.Background(), json.RawMessage(`{"timezone":"UTC","format":"2006-01-02T15:04:05Z07:00"}`))
	if err != nil {
		t.Fatalf("now error: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		t.Fatalf("now output parse error: %v (value=%q)", err, s)
	}
}
