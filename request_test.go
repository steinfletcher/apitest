package apitest

import (
	"bytes"
	"errors"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type failingReadFS struct{}

func (failingReadFS) Open(string) (fs.File, error) { return failingFile{}, nil }

type failingFile struct{}

func (failingFile) Stat() (fs.FileInfo, error) { return nil, errors.New("stat failed") }
func (failingFile) Read([]byte) (int, error)   { return 0, errors.New("read failed") }
func (failingFile) Close() error               { return nil }

func withFailingMultipartWriter(r *Request) *Request {
	r.multipartBody = &bytes.Buffer{}
	r.multipart = multipart.NewWriter(failingWriter{})
	return r
}

func TestRequest_MultipartFormDataReportsWriteErrors(t *testing.T) {
	request := withFailingMultipartWriter(New().Post("/upload"))

	request.MultipartFormData("field", "value")

	if request.err == nil || !strings.Contains(request.err.Error(), "write failed") {
		t.Fatalf("expected the write error to be recorded, got %v", request.err)
	}
}

func TestRequest_MultipartFileReportsPartErrors(t *testing.T) {
	request := withFailingMultipartWriter(New().Post("/upload"))

	request.MultipartFile("file", "testdata/request_body.json")

	if request.err == nil || !strings.Contains(request.err.Error(), "write failed") {
		t.Fatalf("expected the part error to be recorded, got %v", request.err)
	}
}

func TestRequest_MultipartFileReportsReadErrors(t *testing.T) {
	request := New().UseFS(failingReadFS{}).Post("/upload")

	request.MultipartFile("file", "unreadable.txt")

	if request.err == nil || !strings.Contains(request.err.Error(), "read failed") {
		t.Fatalf("expected the read error to be recorded, got %v", request.err)
	}
}

func TestRequest_FailReportsImmediatelyOnceExpectHasBeenCalled(t *testing.T) {
	rec := &recordingT{}
	request := New().Post("/upload")
	request.Expect(rec)

	request.BodyFromFile("testdata/does-not-exist.json")

	if len(rec.fatals) != 1 || !strings.Contains(rec.fatals[0], "does-not-exist.json") {
		t.Fatalf("expected an immediate fatal, got %q", rec.fatals)
	}
	if request.err != nil {
		t.Fatal("expected the error not to be deferred")
	}
}

func TestRequest_BuildReportsMultipartCloseErrors(t *testing.T) {
	rec := &recordingT{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	request := withFailingMultipartWriter(New().Handler(handler).Post("/upload"))

	request.Expect(rec).Status(http.StatusOK).End()

	if len(rec.fatals) != 1 || !strings.Contains(rec.fatals[0], "write failed") {
		t.Fatalf("expected the close error to be fatal, got %q", rec.fatals)
	}
}
