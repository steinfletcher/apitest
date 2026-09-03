package apitest

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiagram_BadgeCSSClass(t *testing.T) {
	tests := []struct {
		status int
		class  string
	}{
		{status: http.StatusOK, class: "badge badge-success"},
		{status: http.StatusInternalServerError, class: "badge badge-danger"},
		{status: http.StatusBadRequest, class: "badge badge-warning"},
	}
	for _, test := range tests {
		t.Run(test.class, func(t *testing.T) {
			class := badgeCSSClass(test.status)

			assert.Equal(t, test.class, class)
		})
	}
}

func TestFormatBodyContent_ShouldReplaceBody(t *testing.T) {
	stream := io.NopCloser(strings.NewReader("lol"))

	val, err := formatBodyContent(stream, func(replacementBody io.ReadCloser) {
		stream = replacementBody
	})
	assert.NoError(t, err)
	assert.Equal(t, "lol", val)

	valSecondRun, errSecondRun := formatBodyContent(stream, func(replacementBody io.ReadCloser) {
		stream = replacementBody
	})
	assert.NoError(t, errSecondRun)
	assert.Equal(t, "lol", valSecondRun)
}

func TestWebSequenceDiagram_GeneratesDSL(t *testing.T) {
	wsd := webSequenceDiagramDSL{}
	wsd.addRequestRow("A", "B", "request1")
	wsd.addRequestRow("B", "C", "request2")
	wsd.addResponseRow("C", "B", "response1")
	wsd.addResponseRow("B", "A", "response2")

	actual := wsd.toString()

	expected := `"A"->"B": (1) request1
"B"->"C": (2) request2
"C"->>"B": (3) response1
"B"->>"A": (4) response2
`
	if expected != actual {
		t.Fatalf("expected=%s != \nactual=%s", expected, actual)
	}
}

func TestNewSequenceDiagramFormatter_SetsDefaultPath(t *testing.T) {
	formatter := SequenceDiagram()

	assert.Equal(t, ".sequence", formatter.storagePath)
}

func TestNewSequenceDiagramFormatter_OverridesPath(t *testing.T) {
	formatter := SequenceDiagram(".sequence-diagram")

	assert.Equal(t, ".sequence-diagram", formatter.storagePath)
}

func TestRecorderBuilder(t *testing.T) {
	recorder := aRecorder()

	assert.Equal(t, 4, len(recorder.Events))
	assert.Equal(t, "title", recorder.Title)
	assert.Equal(t, "subTitle", recorder.SubTitle)
	assert.Equal(t, map[string]any{
		"path":   "/user",
		"name":   "some test",
		"host":   "example.com",
		"method": "GET",
	}, recorder.Meta)
	assert.Equal(t, "reqSource", recorder.Events[0].(HttpRequest).Source)
	assert.Equal(t, "mesReqSource", recorder.Events[1].(MessageRequest).Source)
	assert.Equal(t, "mesResSource", recorder.Events[2].(MessageResponse).Source)
	assert.Equal(t, "resSource", recorder.Events[3].(HttpResponse).Source)
}

func TestNewHTMLTemplateModel_ErrorsIfNoEventsDefined(t *testing.T) {
	recorder := NewTestRecorder()

	_, err := newHTMLTemplateModel(recorder)

	assert.Equal(t, "no events are defined", err.Error())
}

func TestNewHTMLTemplateModel_Success(t *testing.T) {
	recorder := aRecorder()

	model, err := newHTMLTemplateModel(recorder)

	assert.True(t, err == nil)
	assert.Equal(t, 4, len(model.LogEntries))
	assert.Equal(t, "title", model.Title)
	assert.Equal(t, "subTitle", model.SubTitle)
	assert.Equal(t, template.JS(`{"host":"example.com","method":"GET","name":"some test","path":"/user"}`), model.MetaJSON)
	assert.Equal(t, http.StatusNoContent, model.StatusCode)
	assert.Equal(t, "badge badge-success", model.BadgeClass)
	assert.True(t, strings.Contains(model.WebSequenceDSL, "GET /abcdef"))
}

func aRecorder() *Recorder {
	return NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddHttpRequest(aRequest()).
		AddMessageRequest(MessageRequest{Header: "A", Body: "B", Source: "mesReqSource"}).
		AddMessageResponse(MessageResponse{Header: "C", Body: "D", Source: "mesResSource"}).
		AddHttpResponse(aResponse()).
		AddMeta(map[string]any{
			"path":   "/user",
			"name":   "some test",
			"host":   "example.com",
			"method": "GET",
		})
}

func TestNewHttpRequestLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`{"a": 12345}`))

	logEntry, err := newHTTPRequestLogEntry(req)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, "GET /path"))
	assert.True(t, strings.Contains(logEntry.Header, "HTTP/1.1"))
	assert.JSONEq(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntry_JSON(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          io.NopCloser(strings.NewReader(`{"a": 12345}`)),
	}

	logEntry, err := newHTTPResponseLogEntry(response)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.JSONEq(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntry_PlainText(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          io.NopCloser(strings.NewReader(`abcdef`)),
	}

	logEntry, err := newHTTPResponseLogEntry(response)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.Equal(t, logEntry.Body, `abcdef`)
}

func aRequest() HttpRequest {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/abcdef?name=abc", nil)
	req.Header.Set("Content-Type", "application/json")
	return HttpRequest{Value: req, Source: "reqSource", Target: "reqTarget"}
}

func aResponse() HttpResponse {
	return HttpResponse{
		Value: &http.Response{
			StatusCode:    http.StatusNoContent,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 0,
		},
		Source: "resSource",
		Target: "resTarget",
	}
}

func TestWebSequenceDiagram_RenamesOnlyExactDefaultParticipantNames(t *testing.T) {
	dsl := &webSequenceDiagramDSL{meta: map[string]any{
		"consumerName":        "consumer",
		"systemUnderTestName": "app",
	}}

	dsl.addRequestRow(ConsumerDefaultName, SystemUnderTestDefaultName, "GET /")
	dsl.addRequestRow(SystemUnderTestDefaultName, "client.example.com", "GET /")
	dsl.addResponseRow("consultant.example.com", SystemUnderTestDefaultName, "200")

	expected := "\"consumer\"->\"app\": (1) GET /\n" +
		"\"app\"->\"client.example.com\": (2) GET /\n" +
		"\"consultant.example.com\"->>\"app\": (3) 200\n"
	assert.Equal(t, expected, dsl.toString())
}

type recordingFileSystem struct {
	created []string
	closed  int
	content strings.Builder
}

func (f *recordingFileSystem) create(name string) (io.WriteCloser, error) {
	f.created = append(f.created, name)
	return &recordingFile{fs: f}, nil
}

func (f *recordingFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return nil
}

type recordingFile struct {
	fs *recordingFileSystem
}

func (r *recordingFile) Write(p []byte) (int, error) {
	return r.fs.content.Write(p)
}

func (r *recordingFile) Close() error {
	r.fs.closed++
	return nil
}

func TestSequenceDiagramFormatter_ClosesTheDiagramFile(t *testing.T) {
	fs := &recordingFileSystem{}
	formatter := &SequenceDiagramFormatter{storagePath: ".sequence", fs: fs}
	recorder := NewTestRecorder().
		AddTitle("title").
		AddMeta(map[string]any{"hash": "abc123"}).
		AddHttpRequest(HttpRequest{
			Source: ConsumerDefaultName,
			Target: SystemUnderTestDefaultName,
			Value:  httptest.NewRequest(http.MethodGet, "/user", nil),
		}).
		AddHttpResponse(HttpResponse{
			Source: SystemUnderTestDefaultName,
			Target: ConsumerDefaultName,
			Value:  &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))},
		})

	formatter.Format(recorder)

	assert.Equal(t, 1, len(fs.created))
	assert.Equal(t, true, strings.HasSuffix(fs.created[0], "abc123.html"))
	assert.Equal(t, 1, fs.closed)
	assert.Equal(t, true, strings.Contains(fs.content.String(), "<!DOCTYPE html>"))
}

type fakeDiagramFileSystem struct {
	mkdirErr  error
	createErr error
	writeErr  error
	content   strings.Builder
}

func (f *fakeDiagramFileSystem) create(name string) (io.WriteCloser, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &fakeDiagramFile{fs: f}, nil
}

func (f *fakeDiagramFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return f.mkdirErr
}

type fakeDiagramFile struct {
	fs *fakeDiagramFileSystem
}

func (f *fakeDiagramFile) Write(p []byte) (int, error) {
	if f.fs.writeErr != nil {
		return 0, f.fs.writeErr
	}
	return f.fs.content.Write(p)
}

func (f *fakeDiagramFile) Close() error { return nil }

func completeRecorder() *Recorder {
	return NewTestRecorder().
		AddTitle("title").
		AddMeta(map[string]any{"hash": "abc123"}).
		AddHttpRequest(HttpRequest{Source: ConsumerDefaultName, Target: SystemUnderTestDefaultName, Value: httptest.NewRequest(http.MethodGet, "/user", nil)}).
		AddHttpResponse(HttpResponse{Source: SystemUnderTestDefaultName, Target: ConsumerDefaultName, Value: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}})
}

func assertPanicContains(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic containing %q", expected)
		}
		if !strings.Contains(fmt.Sprint(r), expected) {
			t.Fatalf("expected panic containing %q, got %v", expected, r)
		}
	}()
	fn()
}

func TestSequenceDiagramFormatter_WritesToTheOSFileSystem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "diagrams")

	SequenceDiagram(dir).Format(completeRecorder())

	content, err := os.ReadFile(filepath.Join(dir, "abc123.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<!DOCTYPE html>") {
		t.Fatal("expected the diagram html to be written")
	}
}

func TestSequenceDiagramFormatter_PanicsOnFailures(t *testing.T) {
	assertPanicContains(t, "no events are defined", func() {
		(&SequenceDiagramFormatter{storagePath: ".sequence", fs: &fakeDiagramFileSystem{}}).Format(NewTestRecorder())
	})
	assertPanicContains(t, "mkdir failed", func() {
		(&SequenceDiagramFormatter{storagePath: ".sequence", fs: &fakeDiagramFileSystem{mkdirErr: errors.New("mkdir failed")}}).Format(completeRecorder())
	})
	assertPanicContains(t, "create failed", func() {
		(&SequenceDiagramFormatter{storagePath: ".sequence", fs: &fakeDiagramFileSystem{createErr: errors.New("create failed")}}).Format(completeRecorder())
	})
	assertPanicContains(t, "write failed", func() {
		(&SequenceDiagramFormatter{storagePath: ".sequence", fs: &fakeDiagramFileSystem{writeErr: errors.New("write failed")}}).Format(completeRecorder())
	})
}

func TestFormatDiagramRequest_TruncatesLongPaths(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/"+strings.Repeat("a", 100)+"?b=1", nil)

	got := formatDiagramRequest(req)

	if len(got) != 68 || !strings.HasSuffix(got, "...") {
		t.Fatalf("expected a truncated description, got %q", got)
	}
}

type customEvent struct{}

func (customEvent) GetTime() time.Time { return time.Time{} }

func TestNewHTMLTemplateModel_Errors(t *testing.T) {
	failingBody := io.NopCloser(failingReader{})

	_, err := newHTMLTemplateModel(NewTestRecorder().AddHttpRequest(HttpRequest{Value: &http.Request{Method: http.MethodGet, URL: mustParseURL("/x"), Body: failingBody}}))
	if err == nil || err.Error() != "read failed" {
		t.Fatalf("expected the request body error, got %v", err)
	}

	_, err = newHTMLTemplateModel(NewTestRecorder().AddHttpResponse(HttpResponse{Value: &http.Response{StatusCode: http.StatusOK, Body: failingBody}}))
	if err == nil || err.Error() != "read failed" {
		t.Fatalf("expected the response body error, got %v", err)
	}

	_, err = newHTMLTemplateModel(NewTestRecorder().AddHttpRequest(HttpRequest{Value: httptest.NewRequest(http.MethodGet, "/x", nil)}))
	if err == nil || err.Error() != "final event should be a response type" {
		t.Fatalf("expected the final event error, got %v", err)
	}

	recorder := completeRecorder()
	recorder.Meta["unmarshallable"] = func() {}
	_, err = newHTMLTemplateModel(recorder)
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected the meta marshalling error, got %v", err)
	}

	assertPanicContains(t, "received unknown event type", func() {
		r := NewTestRecorder()
		r.Events = append(r.Events, customEvent{})
		_, _ = newHTMLTemplateModel(r)
	})
}
