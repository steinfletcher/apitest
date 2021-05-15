package apitest

import (
	"net/http"
	"testing"
)

func TestRecorder_ResponseStatus_RecordsFinalResponseStatus(t *testing.T) {
	status, err := NewTestRecorder().
		AddHttpRequest(HttpRequest{}).
		AddHttpResponse(HttpResponse{Value: &http.Response{StatusCode: http.StatusAccepted}}).
		AddHttpRequest(HttpRequest{}).
		AddHttpResponse(HttpResponse{Value: &http.Response{StatusCode: http.StatusBadRequest}}).
		ResponseStatus()

	assert.Equal(t, true, err == nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRecorder_ResponseStatus_ErrorsIfNoEventsDefined(t *testing.T) {
	_, err := NewTestRecorder().
		ResponseStatus()

	assert.Equal(t, "no events are defined", err.Error())
}

func TestRecorder_ResponseStatus_ErrorsIfFinalEventNotAResponse(t *testing.T) {
	_, err := NewTestRecorder().
		AddHttpRequest(HttpRequest{}).
		ResponseStatus()

	assert.Equal(t, "final event should be a response type", err.Error())
}

func TestRecorder_ResponseStatus_HandlesEventTypes(t *testing.T) {
	rec := NewTestRecorder().
		AddMessageRequest(MessageRequest{}).
		AddMessageResponse(MessageResponse{})

	status, _ := rec.ResponseStatus()
	assert.Equal(t, -1, status)
	assert.Equal(t, 2, len(rec.Events))
}

func TestRecorder_AddsTitle(t *testing.T) {
	rec := NewTestRecorder().
		AddTitle("title")

	assert.Equal(t, rec.Title, "title")
}

func TestRecorder_AddsSubTitle(t *testing.T) {
	rec := NewTestRecorder().
		AddSubTitle("subTitle")

	assert.Equal(t, rec.SubTitle, "subTitle")
}

func TestRecorder_Reset(t *testing.T) {
	meta := map[string]interface{}{
		"test": "meta",
	}
	rec := NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddMeta(meta).
		AddMessageRequest(MessageRequest{})

	rec.Reset()

	assert.Equal(t, &Recorder{}, rec)
}
