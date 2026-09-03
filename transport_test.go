package apitest

import (
	"net/http"
	"testing"
)

func TestBuildResponseFromMock_Nil(t *testing.T) {
	if buildResponseFromMock(nil) != nil {
		t.Fatal("expected a nil mock response to build as nil")
	}
}

func TestTimeoutError(t *testing.T) {
	err := timeoutError{}
	if err.Error() != "deadline exceeded" || !err.Timeout() || !err.Temporary() {
		t.Fatalf("unexpected timeout error %v", err)
	}
}

func TestTransport_DebugLogsUnmatchedRequests(t *testing.T) {
	transport := newTransport([]*Mock{NewMock().Get("http://example.com/other").RespondWith().Status(http.StatusOK).End()}, nil, true, false, nil, nil)

	res, err := transport.RoundTrip(&http.Request{Method: http.MethodGet, URL: mustParseURL("http://example.com/path"), Header: http.Header{}})

	if res != nil || err == nil {
		t.Fatalf("expected an unmatched request to fail, got %v %v", res, err)
	}
}
