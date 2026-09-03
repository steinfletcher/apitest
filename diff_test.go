package apitest

import (
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	if diff(nil, "a") != "" || diff("a", nil) != "" {
		t.Fatal("expected no diff for nil values")
	}
	if diff(1, "a") != "" {
		t.Fatal("expected no diff for values of different types")
	}
	if diff(1, 2) != "" {
		t.Fatal("expected no diff for scalars")
	}
	if got := diff("line one\nline two\n", "line one\nline three\n"); !strings.Contains(got, "-line two") || !strings.Contains(got, "+line three") {
		t.Fatalf("expected a unified diff of the strings, got %q", got)
	}
	if got := diff(map[string]int{"a": 1}, map[string]int{"a": 2}); !strings.Contains(got, "Diff:") || !strings.Contains(got, "(int) 1") || !strings.Contains(got, "(int) 2") {
		t.Fatalf("expected a diff of the dumped maps, got %q", got)
	}
	if got := diff(&[]int{1}, &[]int{2}); !strings.Contains(got, "Diff:") {
		t.Fatalf("expected pointers to be dereferenced, got %q", got)
	}
}
