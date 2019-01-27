package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
)

type WebSequenceDiagram struct {
	data  bytes.Buffer
	count int
}

func (r *WebSequenceDiagram) AddRequestRow(source, target, description string) {
	r.addRow("->", source, target, description)
}

func (r *WebSequenceDiagram) AddResponseRow(source, target, description string) {
	r.addRow("->>", source, target, description)
}

func (r *WebSequenceDiagram) addRow(operation, source, target, description string) {
	r.count += 1
	r.data.WriteString(fmt.Sprintf("%s%s%s: (%d) %s\n",
		source,
		operation,
		target,
		r.count,
		description))
}

func (r *WebSequenceDiagram) ToString() string {
	return r.data.String()
}

type (
	HTMLTemplateModel struct {
		Title          string
		SubTitle       string
		StatusCode     int
		BadgeClass     string
		LogEntries     []LogEntry
		WebSequenceDSL string
		MetaJSON       template.JS
	}

	LogEntry struct {
		Header string
		Body   string
	}
)

type SequenceDiagramFormatter struct {
	storagePath string
}

func (r *SequenceDiagramFormatter) Format(recorder *Recorder) {
	output, err := NewHTMLTemplateModel(recorder)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("sequenceDiagram").
		Funcs(*incTemplateFunc).
		Parse(Template)
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, output)
	if err != nil {
		panic(err)
	}

	fileName := "fixmelater.html"
	err = os.MkdirAll(".sequence", os.ModePerm)
	if err != nil {
		panic(err)
	}
	saveFilesTo := fmt.Sprintf("%s/.sequence/%s", r.storagePath, fileName)

	f, err := os.Create(saveFilesTo)
	if err != nil {
		panic(err)
	}

	s, _ := filepath.Abs(saveFilesTo)
	_, err = f.WriteString(out.String())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created sequence diagram (%s): file://%s\n", fileName, s)
}

func NewSequenceDiagramFormatter(path ...string) *SequenceDiagramFormatter {
	var storagePath string
	if len(path) == 0 {
		storagePath = "."
	} else {
		storagePath = path[0]
	}
	return &SequenceDiagramFormatter{storagePath: storagePath}
}

var incTemplateFunc = &template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
}

func badgeCSSClass(status int) string {
	class := "badge badge-success"
	if status >= 400 && status < 500 {
		class = "badge badge-warning"
	} else if status >= 500 {
		class = "badge badge-danger"
	}
	return class
}

func NewHTMLTemplateModel(r *Recorder) (HTMLTemplateModel, error) {
	if len(r.Events) == 0 {
		return HTMLTemplateModel{}, errors.New("no events are defined")
	}
	var logs []LogEntry
	webSequenceDiagram := &WebSequenceDiagram{}

	for _, event := range r.Events {
		switch v := event.(type) {
		case HttpRequest:
			httpReq := v.Value
			webSequenceDiagram.AddRequestRow(v.Source, v.Target, fmt.Sprintf("%s %s", httpReq.Method, httpReq.URL))
			entry, err := newHttpRequestLogEntry(httpReq)
			if err != nil {
				return HTMLTemplateModel{}, err
			}
			logs = append(logs, entry)
		case HttpResponse:
			webSequenceDiagram.AddResponseRow(v.Source, v.Target, strconv.Itoa(v.Value.StatusCode))
			entry, err := newHttpResponseLogEntry(v.Value)
			if err != nil {
				return HTMLTemplateModel{}, err
			}
			logs = append(logs, entry)
		case MessageRequest:
			webSequenceDiagram.AddRequestRow(v.Source, v.Target, v.Header)
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body})
		case MessageResponse:
			webSequenceDiagram.AddResponseRow(v.Source, v.Target, v.Header)
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body})
		default:
			panic("received unknown event type")
		}
	}

	status, err := r.ResponseStatus()
	if err != nil {
		return HTMLTemplateModel{}, err
	}

	jsonMeta, err := json.Marshal(r.Meta)
	if err != nil {
		return HTMLTemplateModel{}, err
	}

	return HTMLTemplateModel{
		WebSequenceDSL: webSequenceDiagram.ToString(),
		LogEntries:     logs,
		Title:          r.Title,
		SubTitle:       r.SubTitle,
		StatusCode:     status,
		BadgeClass:     badgeCSSClass(status),
		MetaJSON:       template.JS(jsonMeta),
	}, nil
}

func newHttpRequestLogEntry(req *http.Request) (LogEntry, error) {
	reqHeader, err := httputil.DumpRequest(req, false)
	if err != nil {
		return LogEntry{}, err
	}
	body, err := formatBodyContent(req.Body)
	if err != nil {
		return LogEntry{}, err
	}
	return LogEntry{Header: string(reqHeader), Body: body}, err
}

func newHttpResponseLogEntry(res *http.Response) (LogEntry, error) {
	resDump, err := httputil.DumpResponse(res, false)
	if err != nil {
		return LogEntry{}, err
	}
	body, err := formatBodyContent(res.Body)
	if err != nil {
		return LogEntry{}, err
	}
	return LogEntry{Header: string(resDump), Body: body}, err
}

func formatBodyContent(bodyReadCloser io.ReadCloser) (string, error) {
	if bodyReadCloser == nil {
		return "", nil
	}

	body, err := ioutil.ReadAll(bodyReadCloser)
	if err != nil {
		return "", err
	}
	bodyReadCloser = ioutil.NopCloser(bytes.NewBuffer(body))

	buf := new(bytes.Buffer)
	if isJSON(string(body)) {
		jsonEncodeErr := json.Indent(buf, body, "", "    ")
		if jsonEncodeErr != nil {
			return "", jsonEncodeErr
		}
		s := buf.String()
		return s, nil
	}

	_, err = buf.Write(body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
