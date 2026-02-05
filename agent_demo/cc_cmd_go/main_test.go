package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		out  []string
	}{
		{name: "empty", in: nil, out: nil},
		{name: "no-dashdash", in: []string{"-p", "hi"}, out: []string{"-p", "hi"}},
		{name: "dashdash", in: []string{"--", "-p", "hi"}, out: []string{"-p", "hi"}},
		{name: "only-dashdash", in: []string{"--"}, out: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeArgs(tc.in)
			if !reflect.DeepEqual(got, tc.out) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tc.in, got, tc.out)
			}
		})
	}
}
