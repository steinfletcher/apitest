package apitest

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCopyHttpResponse_DoesNotShareHeaderValuesWithTheOriginal(t *testing.T) {
	original := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Custom": {"a"}},
		Body:       io.NopCloser(strings.NewReader("body")),
	}

	copied := copyHttpResponse(original)
	copied.Header["X-Custom"][0] = "changed"
	copied.Header.Add("X-Custom", "b")

	assert.Equal(t, []string{"a"}, original.Header["X-Custom"])
	assert.Equal(t, []string{"changed", "b"}, copied.Header["X-Custom"])

	copiedBody, _ := io.ReadAll(copied.Body)
	originalBody, _ := io.ReadAll(original.Body)
	assert.Equal(t, "body", string(copiedBody))
	assert.Equal(t, "body", string(originalBody))
}

func TestMockInteraction_GetRequestHostFallsBackToTheURL(t *testing.T) {
	interaction := &mockInteraction{request: &http.Request{URL: mustParseURL("http://users.example.com/path")}}
	if interaction.GetRequestHost() != "users.example.com" {
		t.Fatalf("unexpected host %q", interaction.GetRequestHost())
	}
}
