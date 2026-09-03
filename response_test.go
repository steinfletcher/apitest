package apitest

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestResponse_BodyFromFileReportsMissingFiles(t *testing.T) {
	rec := &recordingT{}

	New().Get("/").Expect(rec).BodyFromFile("testdata/does-not-exist.json")

	if len(rec.fatals) != 1 || !strings.Contains(rec.fatals[0], "does-not-exist.json") {
		t.Fatalf("expected a fatal, got %q", rec.fatals)
	}
}

func TestResult_JSONPanicsOnBadBodies(t *testing.T) {
	assertPanics := func(name string, body io.Reader) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected a panic", name)
			}
		}()
		var v map[string]any
		Result{Response: &http.Response{Body: io.NopCloser(body)}}.JSON(&v)
	}
	assertPanics("unreadable body", failingReader{})
	assertPanics("invalid json", strings.NewReader("not json"))
}
