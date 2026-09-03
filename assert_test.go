package apitest

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestApiTest_Assert_StatusCodes(t *testing.T) {
	tests := []struct {
		responseStatus []int
		assertFunc     Assert
		isSuccess      bool
	}{
		{[]int{200, 312, 399}, IsSuccess, true},
		{[]int{400, 404, 499}, IsClientError, true},
		{[]int{500, 503}, IsServerError, true},
		{[]int{400, 500}, IsSuccess, false},
		{[]int{200, 500}, IsClientError, false},
		{[]int{200, 400}, IsServerError, false},
	}
	for _, test := range tests {
		for _, status := range test.responseStatus {
			response := &http.Response{StatusCode: status}
			err := test.assertFunc(response, nil)
			if test.isSuccess && err != nil {
				t.Fatalf("Expecteted nil but received %s", err)
			} else if !test.isSuccess && err == nil {
				t.Fatalf("Expected error but didn't receive one")
			}
		}
	}
}

type recordingT struct {
	errors []string
	fatals []string
}

func (r *recordingT) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recordingT) Fatal(args ...any) {
	r.fatals = append(r.fatals, fmt.Sprint(args...))
}

func (r *recordingT) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func TestDefaultVerifier_KeepsMessageWhenTestIsUnnamed(t *testing.T) {
	content := messageFromMsgAndArgs("Status code 200 not equal to 201", failureMessageArgs{Name: ""})

	if len(content) != 1 || content[0].label != "Messages" || content[0].content != "Status code 200 not equal to 201" {
		t.Fatalf("expected the message to be kept, got %#v", content)
	}
}

func TestDefaultVerifier_KeepsMessageAndNameWhenTestIsNamed(t *testing.T) {
	content := messageFromMsgAndArgs("Status code 200 not equal to 201", failureMessageArgs{Name: "my test"})

	if len(content) != 2 || content[0].content != "Status code 200 not equal to 201" || content[1].content != "my test" {
		t.Fatalf("expected the message and name to be kept, got %#v", content)
	}
}

func TestDefaultVerifier_FailReportsMessageForUnnamedTest(t *testing.T) {
	rec := &recordingT{}

	DefaultVerifier{}.Equal(rec, 201, 200, "Status code 200 not equal to 201", failureMessageArgs{})

	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "Status code 200 not equal to 201") {
		t.Fatalf("expected the failure output to contain the message, got %q", rec.errors)
	}
}

type namedRecordingT struct {
	recordingT
}

func (namedRecordingT) Name() string { return "the test name" }

func TestDefaultVerifier_True(t *testing.T) {
	rec := &recordingT{}

	if !(DefaultVerifier{}).True(rec, true) {
		t.Fatal("expected true to pass")
	}
	if (DefaultVerifier{}).True(rec, false, "custom message") {
		t.Fatal("expected false to fail")
	}
	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "Should be true") || !strings.Contains(rec.errors[0], "custom message") {
		t.Fatalf("unexpected failure output %q", rec.errors)
	}
}

func TestDefaultVerifier_JSONEq(t *testing.T) {
	tests := map[string]struct {
		expected, actual string
		wantErr          string
	}{
		"equivalent":       {`{"a": 1, "b": [1, 2]}`, `{"b":[1,2],"a":1}`, ""},
		"different":        {`{"a": 1}`, `{"a": 2}`, "Not equal"},
		"invalid expected": {`{`, `{}`, "Expected value ('{') is not valid json"},
		"invalid actual":   {`{}`, `nope`, "Input ('nope') needs to be valid json"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rec := &recordingT{}
			ok := (DefaultVerifier{}).JSONEq(rec, test.expected, test.actual)
			if test.wantErr == "" {
				if !ok || len(rec.errors) != 0 {
					t.Fatalf("expected success, got %q", rec.errors)
				}
				return
			}
			if ok || len(rec.errors) != 1 || !strings.Contains(rec.errors[0], test.wantErr) {
				t.Fatalf("expected failure containing %q, got ok=%v %q", test.wantErr, ok, rec.errors)
			}
		})
	}
}

func TestDefaultVerifier_NoError(t *testing.T) {
	rec := &recordingT{}
	if !(DefaultVerifier{}).NoError(rec, nil) {
		t.Fatal("expected nil error to pass")
	}
	if (DefaultVerifier{}).NoError(rec, errors.New("boom")) || !strings.Contains(rec.errors[0], "Received unexpected error:\n") {
		t.Fatalf("expected an error to fail, got %q", rec.errors)
	}
}

func TestDefaultVerifier_Equal(t *testing.T) {
	tests := map[string]struct {
		expected, actual any
		wantErr          string
	}{
		"equal":               {1, 1, ""},
		"equal bytes":         {[]byte("a"), []byte("a"), ""},
		"nil bytes":           {[]byte(nil), []byte(nil), ""},
		"both nil":            {nil, nil, ""},
		"nil and value":       {nil, 1, "Not equal"},
		"bytes and string":    {[]byte("a"), "a", "Not equal"},
		"bytes and nil":       {[]byte("a"), []byte(nil), "Not equal"},
		"different types":     {1, "1", "expected: int(1)"},
		"durations":           {time.Second, time.Minute, "expected: 1s"},
		"function argument":   {func() {}, 1, "Invalid operation"},
		"functions both":      {func() {}, func() {}, "cannot take func type as argument"},
		"structured mismatch": {map[string]int{"a": 1}, map[string]int{"a": 2}, "Diff:"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rec := &recordingT{}
			ok := (DefaultVerifier{}).Equal(rec, test.expected, test.actual)
			if test.wantErr == "" {
				if !ok || len(rec.errors) != 0 {
					t.Fatalf("expected success, got %q", rec.errors)
				}
				return
			}
			if ok || len(rec.errors) != 1 || !strings.Contains(rec.errors[0], test.wantErr) {
				t.Fatalf("expected failure containing %q, got ok=%v %q", test.wantErr, ok, rec.errors)
			}
		})
	}
}

func TestDefaultVerifier_FailIncludesTestNameWhenAvailable(t *testing.T) {
	rec := &namedRecordingT{}

	(DefaultVerifier{}).Fail(rec, "boom", 42)

	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "Test:") || !strings.Contains(rec.errors[0], "the test name") || !strings.Contains(rec.errors[0], "Messages:") {
		t.Fatalf("unexpected failure output %q", rec.errors)
	}
}

func TestDefaultVerifier_FailFromAGoroutine(t *testing.T) {
	rec := &recordingT{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		(DefaultVerifier{}).Fail(rec, "boom")
	}()
	<-done

	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "Error Trace:") {
		t.Fatalf("unexpected failure output %q", rec.errors)
	}
}

func TestAssertHelpers(t *testing.T) {
	long := strings.Repeat("x", bufio.MaxScanTokenSize)
	if got := truncatingFormat(long); !strings.HasSuffix(got, "<... truncated>") || len(got) > bufio.MaxScanTokenSize {
		t.Fatalf("expected a truncated value, got %d bytes", len(got))
	}
	if truncatingFormat("short") != `"short"` {
		t.Fatal("expected short values to be untouched")
	}

	if isFunction(nil) || !isFunction(func() {}) || isFunction(1) {
		t.Fatal("isFunction misclassified a value")
	}
	if err := validateEqualArgs(nil, nil); err != nil {
		t.Fatal(err)
	}

	if !isTest("Test", "Test") || !isTest("TestX", "Test") || isTest("Testx", "Test") || isTest("Bench", "Test") {
		t.Fatal("isTest misclassified a name")
	}

	if got := messageFromMsgAndArgs(); got != nil {
		t.Fatalf("expected nil for no args, got %#v", got)
	}
	if got := messageFromMsgAndArgs("only"); len(got) != 1 || got[0].content != "only" {
		t.Fatalf("unexpected %#v", got)
	}
	if got := messageFromMsgAndArgs(failureMessageArgs{}); got != nil {
		t.Fatalf("expected nil for an unnamed test, got %#v", got)
	}
	if got := messageFromMsgAndArgs(failureMessageArgs{Name: "n"}); len(got) != 1 || got[0].label != "Name" {
		t.Fatalf("unexpected %#v", got)
	}
	if got := messageFromMsgAndArgs(42); len(got) != 1 || got[0].content != "42" {
		t.Fatalf("unexpected %#v", got)
	}
	if got := messageFromMsgAndArgs(1, 2); len(got) != 0 {
		t.Fatalf("expected no content for non-string args, got %#v", got)
	}
}

func TestNoopVerifier_PassesEverything(t *testing.T) {
	rec := &recordingT{}
	v := NoopVerifier{}
	if !v.True(rec, false) || !v.Fail(rec, "boom") || !v.NoError(rec, errors.New("boom")) || !v.Equal(rec, 1, 2) || !v.JSONEq(rec, "{", "}") {
		t.Fatal("expected every assertion to pass")
	}
	if len(rec.errors)+len(rec.fatals) != 0 {
		t.Fatalf("expected no reports, got %q %q", rec.errors, rec.fatals)
	}
}

func TestDefaultVerifier_FailThroughAPointerWrapper(t *testing.T) {
	rec := &recordingT{}
	var verifier Verifier = &DefaultVerifier{}

	verifier.Fail(rec, "boom")

	if len(rec.errors) != 1 || !strings.Contains(rec.errors[0], "boom") {
		t.Fatalf("unexpected failure output %q", rec.errors)
	}
}
