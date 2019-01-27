package apitest

import (
	"github.com/steinfletcher/api-test/assert"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func TestWebSequenceDiagram_GeneratesDSL(t *testing.T) {
	wsd := WebSequenceDiagram{}
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

func TestRecorderBuilder(t *testing.T) {
	recorder := aRecorder()

	assert.Len(t, recorder.Events, 4)
	assert.Equal(t, "title", recorder.Title)
	assert.Equal(t, "subTitle", recorder.SubTitle)
	assert.Equal(t, map[string]interface{}{"Z": "1"}, recorder.Meta)
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
	assert.Equal(t, template.JS(`{"Z":"1"}`), model.MetaJSON)
	assert.Equal(t, http.StatusNoContent, model.StatusCode)
	assert.Equal(t, "badge badge-success", model.BadgeClass)
}

func aRecorder() *Recorder {
	return NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddHttpRequest(aRequest()).
		AddMessageRequest(MessageRequest{Header: "A", Body: "B", Source: "mesReqSource"}).
		AddMessageResponse(MessageResponse{Header: "C", Body: "D", Source: "mesResSource"}).
		AddHttpResponse(aResponse()).
		AddMetaJSON(map[string]interface{}{"Z": "1"})
}

func TestNewHttpRequestLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`{"a": 12345}`))

	logEntry, err := newHttpRequestLogEntry(req)

	assert.Nil(t, err)
	assert.True(t, strings.Contains(logEntry.Header, "GET /path"))
	assert.True(t, strings.Contains(logEntry.Header, "HTTP/1.1"))
	assert.JsonEqual(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntry_JSON(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          ioutil.NopCloser(strings.NewReader(`{"a": 12345}`)),
	}

	logEntry, err := newHttpResponseLogEntry(response)

	assert.Nil(t, err)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.JsonEqual(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntry_PlainText(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          ioutil.NopCloser(strings.NewReader(`abcdef`)),
	}

	logEntry, err := newHttpResponseLogEntry(response)

	assert.Nil(t, err)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.Equal(t, logEntry.Body, `abcdef`)
}

func aRequest() HttpRequest {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/abcdef", nil)
	req.Header.Set("Content-Type", "application/json")
	return HttpRequest{Value: req, Source: "reqSource", Target: "reqTarget"}
}

func aResponse() HttpResponse {
	return HttpResponse{
		Value: &http.Response{
			StatusCode: http.StatusNoContent,
		},
		Source: "resSource",
		Target: "resTarget",
	}
}
