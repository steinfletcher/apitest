package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
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
