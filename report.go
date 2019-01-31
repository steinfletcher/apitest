package apitest

import (
	"errors"
	"net/http"
)

type (
	ReportFormatter interface {
		Format(*Recorder)
	}

	Recorder struct {
		Title    string
		SubTitle string
		Meta     map[string]interface{}
		Events   []interface{}
	}

	MessageRequest struct {
		Source string
		Target string
		Header string
		Body   string
	}

	MessageResponse struct {
		Source string
		Target string
		Header string
		Body   string
	}

	HttpRequest struct {
		Source string
		Target string
		Value  *http.Request
	}

	HttpResponse struct {
		Source string
		Target string
		Value  *http.Response
	}
)

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
