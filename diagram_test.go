package apitest

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
		AddMeta(map[string]interface{}{
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

type FS struct {
	CapturedCreateName   string
	CapturedCreateFile   string
	CapturedMkdirAllPath string
}

func (m *FS) create(name string) (*os.File, error) {
	m.CapturedCreateName = name
	file, err := os.CreateTemp("/tmp", "apitest")
	if err != nil {
		panic(err)
	}
	m.CapturedCreateFile = file.Name()
	return file, nil
}

func (m *FS) mkdirAll(path string, perm os.FileMode) error {
	m.CapturedMkdirAllPath = path
	return nil
}
