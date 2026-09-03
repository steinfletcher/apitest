package difflib

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// failAfterWriter fails the nth call to Write.
type failAfterWriter struct {
	failOn int
	calls  int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failOn {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func huge(prefix string) string {
	return prefix + strings.Repeat("x", 5000)
}

func TestWriteUnifiedDiff_ReportsWriterErrors(t *testing.T) {
	diff := UnifiedDiff{
		A:        []string{huge("a"), huge("b"), huge("c")},
		B:        []string{huge("a"), huge("y"), huge("c")},
		FromFile: huge("from"),
		ToFile:   huge("to"),
		Eol:      huge("\n"),
		Context:  3,
	}

	// headers, hunk header, context line, deletion, insertion
	for failOn := 1; failOn <= 6; failOn++ {
		w := &failAfterWriter{failOn: failOn}
		if err := WriteUnifiedDiff(w, diff); err == nil || err.Error() != "write failed" {
			t.Fatalf("write %d: expected the writer error, got %v", failOn, err)
		}
	}

	if err := WriteUnifiedDiff(&failAfterWriter{failOn: 100}, diff); err != nil {
		t.Fatal(err)
	}
}

func TestWriteContextDiff_ReportsWriterErrors(t *testing.T) {
	base := ContextDiff{
		A:       []string{"a\n", "b\n", "c\n"},
		B:       []string{"a\n", "y\n", "c\n"},
		Context: 3,
	}

	viaHeader := base
	viaHeader.FromFile = huge("from")
	if err := WriteContextDiff(&failAfterWriter{failOn: 1}, viaHeader); err == nil || err.Error() != "write failed" {
		t.Fatalf("expected the header write error, got %v", err)
	}

	viaLines := base
	viaLines.A = []string{huge("a"), huge("b"), huge("c")}
	viaLines.B = []string{huge("a"), huge("y"), huge("c")}
	if err := WriteContextDiff(&failAfterWriter{failOn: 1}, viaLines); err == nil || err.Error() != "write failed" {
		t.Fatalf("expected the line write error, got %v", err)
	}
}

func TestWriteContextDiff_DefaultsEolAndHandlesInsertsAndDeletes(t *testing.T) {
	var out strings.Builder
	err := WriteContextDiff(&out, ContextDiff{
		A:       SplitLines("a\nb\nc\nd\ne\nf\ng\n"),
		B:       SplitLines("a\nc\nd\ne\nf\nX\ng\n"),
		Context: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, expected := range []string{"- b\n", "+ X\n", "***************", "*** 1,8 ****", "--- 1,8 ----"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected %q in:\n%s", expected, got)
		}
	}
}

func TestColorizeCanBeDisabledByEnvironment(t *testing.T) {
	t.Setenv("COLORIZE_LOGS", "false")
	defer func() { colorizeLog = true }()

	got, err := GetUnifiedDiffString(UnifiedDiff{A: SplitLines("a\n"), B: SplitLines("b\n"), FromFile: "x", ToFile: "y", Context: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "\033[") {
		t.Fatalf("expected no colour codes, got %q", got)
	}
}

func TestSequenceMatcher_AutoJunkPurgesPopularElements(t *testing.T) {
	var b []string
	for i := range 250 {
		if i%20 == 0 {
			b = append(b, "popular")
		} else {
			b = append(b, fmt.Sprintf("line %d", i))
		}
	}
	a := append([]string{}, b...)
	a[100] = "changed"

	m := NewMatcher(a, b)

	if _, ok := m.bPopular["popular"]; !ok {
		t.Fatal("expected the repeated element to be treated as popular")
	}
	if ratio := m.Ratio(); ratio < 0.99 {
		t.Fatalf("expected the sequences to still match closely, got %f", ratio)
	}
	if first, second := m.GetOpCodes(), m.GetOpCodes(); len(first) != len(second) {
		t.Fatal("expected op codes to be cached")
	}
}

func TestSequenceMatcher_ExtendsMatchesOverJunk(t *testing.T) {
	isJunk := func(s string) bool { return s == " " }
	a := []string{" ", "x", " ", "y", " ", "z"}
	b := []string{" ", "x", " ", "y", " ", "z"}

	m := NewMatcherWithJunk(a, b, true, isJunk)

	blocks := m.GetMatchingBlocks()
	if len(blocks) == 0 || blocks[0].Size != len(a) {
		t.Fatalf("expected one full match including the junk, got %+v", blocks)
	}
}

func TestSequenceMatcher_StopsAtTheEndOfTheWindow(t *testing.T) {
	// "q" occurs in b only after the window searched to the left of the longest match
	a := []string{"q", "M", "M", "M", "r"}
	b := []string{"r", "M", "M", "M", "q"}

	blocks := NewMatcher(a, b).GetMatchingBlocks()

	if len(blocks) != 2 || blocks[0] != (Match{A: 1, B: 1, Size: 3}) {
		t.Fatalf("expected only the M M M block to match, got %+v", blocks)
	}
}
