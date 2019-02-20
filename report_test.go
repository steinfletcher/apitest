package apitest

import (
	"github.com/stretchr/testify/assert"
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

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRecorder_ResponseStatus_ErrorsIfNoEventsDefined(t *testing.T) {
	_, err := NewTestRecorder().
		ResponseStatus()

	assert.Error(t, err, "no events are defined")
}

func TestRecorder_ResponseStatus_ErrorsIfFinalEventNotAResponse(t *testing.T) {
	_, err := NewTestRecorder().
		AddHttpRequest(HttpRequest{}).
		ResponseStatus()

	assert.Error(t, err, "final event should be a response type")
}

func TestRecorder_ResponseStatus_HandlesEventTypes(t *testing.T) {
	rec := NewTestRecorder().
		AddMessageRequest(MessageRequest{}).
		AddMessageResponse(MessageResponse{})

	status, _ := rec.ResponseStatus()
	assert.Equal(t, -1, status)
	assert.Len(t, rec.Events, 2)
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
