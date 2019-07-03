package apitest

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSequenceDiagramFormatter_Format(t *testing.T) {
	mockFS := &FS{}
	formatter := SequenceDiagramFormatter{storagePath: ".sequence", fs: mockFS}

	formatter.Format(aRecorder())

	assert.Equal(t, ".sequence", mockFS.CapturedMkdirAllPath)
	assert.True(t, strings.HasSuffix(mockFS.CapturedCreateName, "html"))

	expected, _ := ioutil.ReadFile("testdata/sequence_diagram_snapshot.html")
	actual, _ := ioutil.ReadFile(mockFS.CapturedCreateFile)

	assert.Equal(t, string(expected), string(actual))
}

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

func TestWebSequenceDiagram_GeneratesDSL(t *testing.T) {
	wsd := WebSequenceDiagramDSL{}
	wsd.AddRequestRow("A", "B", "request1")
	wsd.AddRequestRow("B", "C", "request2")
	wsd.AddResponseRow("C", "B", "response1")
	wsd.AddResponseRow("B", "A", "response2")

	actual := wsd.ToString()

	expected := "A->B: (1) request1\nB->C: (2) request2\nC->>B: (3) response1\nB->>A: (4) response2\n"
	if expected != actual {
		t.Fatalf("expected=%s != actual=%s", expected, actual)
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

	assert.Len(t, recorder.Events, 4)
	assert.Equal(t, "title", recorder.Title)
	assert.Equal(t, "subTitle", recorder.SubTitle)
	assert.Equal(t, map[string]interface{}{
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

	_, err := NewHTMLTemplateModel(recorder)

	assert.Error(t, err, "no events are defined")
}

func TestNewHTMLTemplateModel_Success(t *testing.T) {
	recorder := aRecorder()

	model, err := NewHTMLTemplateModel(recorder)

	assert.Nil(t, err)
	assert.Len(t, model.LogEntries, 4)
	assert.Equal(t, "title", model.Title)
	assert.Equal(t, "subTitle", model.SubTitle)
	assert.Equal(t, template.JS(`{"host":"example.com","method":"GET","name":"some test","path":"/user"}`), model.MetaJSON)
	assert.Equal(t, http.StatusNoContent, model.StatusCode)
	assert.Equal(t, "badge badge-success", model.BadgeClass)
	assert.Contains(t, model.WebSequenceDSL, "GET /abcdef")
}

func aRecorder() *Recorder {
	return NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddHttpRequest(aRequest()).
		AddMessageRequest(MessageRequest{Header: "A", Body: "B", Source: "mesReqSource"}).
		AddMessageResponse(MessageResponse{Header: "C", Body: "D", Source: "mesResSource"}).
		AddHttpResponse(aResponse()).
		AddMeta(map[string]interface{}{
			"path":   "/user",
			"name":   "some test",
			"host":   "example.com",
			"method": "GET",
		})
}

func TestNewHttpRequestLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`{"a": 12345}`))

	logEntry, err := NewHttpRequestLogEntry(req)

	assert.Nil(t, err)
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
		Body:          ioutil.NopCloser(strings.NewReader(`{"a": 12345}`)),
	}

	logEntry, err := NewHttpResponseLogEntry(response)

	assert.Nil(t, err)
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
		Body:          ioutil.NopCloser(strings.NewReader(`abcdef`)),
	}

	logEntry, err := NewHttpResponseLogEntry(response)

	assert.Nil(t, err)
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

type FS struct {
	CapturedCreateName   string
	CapturedCreateFile   string
	CapturedMkdirAllPath string
}

func (m *FS) Create(name string) (*os.File, error) {
	m.CapturedCreateName = name
	file, err := ioutil.TempFile("/tmp", "apitest")
	if err != nil {
		panic(err)
	}
	m.CapturedCreateFile = file.Name()
	return file, nil
}

func (m *FS) MkdirAll(path string, perm os.FileMode) error {
	m.CapturedMkdirAllPath = path
	return nil
}
