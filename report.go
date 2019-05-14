package apitest

import (
	"errors"
	"net/http"
	"time"
)

type (
	ReportFormatter interface {
		Format(*Recorder)
	}

	Event interface {
		GetTime() time.Time
	}

	Recorder struct {
		Title    string
		SubTitle string
		Meta     map[string]interface{}
		Events   []Event
	}

	MessageRequest struct {
		Source    string
		Target    string
		Header    string
		Body      string
		Timestamp time.Time
	}

	MessageResponse struct {
		Source    string
		Target    string
		Header    string
		Body      string
		Timestamp time.Time
	}

	HttpRequest struct {
		Source    string
		Target    string
		Value     *http.Request
		Timestamp time.Time
	}

	HttpResponse struct {
		Source    string
		Target    string
		Value     *http.Response
		Timestamp time.Time
	}
)

func (r HttpRequest) GetTime() time.Time     { return r.Timestamp }
func (r HttpResponse) GetTime() time.Time    { return r.Timestamp }
func (r MessageRequest) GetTime() time.Time  { return r.Timestamp }
func (r MessageResponse) GetTime() time.Time { return r.Timestamp }

func NewTestRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) AddHttpRequest(req HttpRequest) *Recorder {
	r.Events = append(r.Events, req)
	return r
}

func (r *Recorder) AddHttpResponse(req HttpResponse) *Recorder {
	r.Events = append(r.Events, req)
	return r
}

func (r *Recorder) AddMessageRequest(m MessageRequest) *Recorder {
	r.Events = append(r.Events, m)
	return r
}

func (r *Recorder) AddMessageResponse(m MessageResponse) *Recorder {
	r.Events = append(r.Events, m)
	return r
}

func (r *Recorder) AddTitle(title string) *Recorder {
	r.Title = title
	return r
}

func (r *Recorder) AddSubTitle(subTitle string) *Recorder {
	r.SubTitle = subTitle
	return r
}

func (r *Recorder) AddMeta(meta map[string]interface{}) *Recorder {
	r.Meta = meta
	return r
}

func (r *Recorder) ResponseStatus() (int, error) {
	if len(r.Events) == 0 {
		return -1, errors.New("no events are defined")
	}

	switch v := r.Events[len(r.Events)-1].(type) {
	case HttpResponse:
		return v.Value.StatusCode, nil
	case MessageResponse:
		return -1, nil
	default:
		return -1, errors.New("final event should be a response type")
	}
}

func (r *Recorder) Reset() {
	r.Title = ""
	r.SubTitle = ""
	r.Events = nil
	r.Meta = nil
}
