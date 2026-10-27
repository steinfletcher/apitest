package apitest

import (
	"encoding/json"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type (
	// reportModel is the data rendered by the sequence diagram report template
	reportModel struct {
		Title        string
		SubTitle     string
		StatusCode   int
		StatusText   string
		StatusClass  string
		Duration     string
		StartedAt    string
		Participants []string
		Events       []reportEvent
		Diagram      htmlTemplate.HTML
		Meta         []metaItem
		MetaJSON     htmlTemplate.JS
	}

	// reportEvent is one interaction in the report, drawn as an arrow in the diagram and listed in the event log
	reportEvent struct {
		Number      int
		Kind        string // request or response
		Source      string
		Target      string
		Label       string
		StartLine   string
		Headers     []headerLine
		Body        string
		BodyHTML    htmlTemplate.HTML
		BodyKind    string // json or text
		BodySize    string
		StatusCode  int
		StatusClass string
		Unanswered  bool // a request that received no response, such as a call that matched no mock
		Offset      string
		Curl        string
		Raw         string
	}

	headerLine struct {
		Name  string
		Value string
	}

	metaItem struct {
		Key   string
		Value string
	}

	// logEntry is the wire representation of an HTTP request or response
	logEntry struct {
		Header    string
		Body      string
		RawBody   string
		Timestamp time.Time
	}

	// SequenceDiagramFormatter implementation of a ReportFormatter
	SequenceDiagramFormatter struct {
		storagePath string
		fs          diagramFileSystem
	}

	diagramFileSystem interface {
		create(name string) (io.WriteCloser, error)
		mkdirAll(path string, perm os.FileMode) error
	}

	osDiagramFileSystem struct{}
)

func (r *osDiagramFileSystem) create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

func (r *osDiagramFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// renameParticipant swaps a default participant name (e.g. "sut") for the user supplied one.
// Only exact matches are renamed so that hosts which merely contain the default name are left alone.
func renameParticipant(participant, defaultName, name string) string {
	if participant == defaultName {
		return name
	}
	return participant
}

// participantNamer applies the consumerName and systemUnderTestName meta values to participant names
type participantNamer map[string]any

func (m participantNamer) name(participant string) string {
	if n, ok := m["consumerName"].(string); ok {
		participant = renameParticipant(participant, ConsumerDefaultName, n)
	}
	if n, ok := m["systemUnderTestName"].(string); ok {
		participant = renameParticipant(participant, SystemUnderTestDefaultName, n)
	}
	return participant
}

// Format formats the events received by the recorder
func (r *SequenceDiagramFormatter) Format(recorder *Recorder) {
	output, err := newHTMLTemplateModel(recorder)
	if err != nil {
		panic(err)
	}

	fileName := fmt.Sprintf("%s.html", recorder.Meta["hash"])
	err = r.fs.mkdirAll(r.storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	saveFilesTo := filepath.Join(r.storagePath, fileName)

	f, err := r.fs.create(saveFilesTo)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := sequenceDiagramTemplate.Execute(f, output); err != nil {
		panic(err)
	}

	s, _ := filepath.Abs(saveFilesTo)
	fmt.Printf("Created sequence diagram (%s): %s\n", fileName, filepath.FromSlash(s))
}

// SequenceDiagram produce a sequence diagram at the given path or .sequence by default
func SequenceDiagram(path ...string) *SequenceDiagramFormatter {
	var storagePath string
	if len(path) == 0 {
		storagePath = ".sequence"
	} else {
		storagePath = path[0]
	}
	return &SequenceDiagramFormatter{storagePath: storagePath, fs: &osDiagramFileSystem{}}
}

var sequenceDiagramTemplate = htmlTemplate.Must(htmlTemplate.New("sequenceDiagram").Parse(reportTemplate))

func formatDiagramRequest(req *http.Request) string {
	out := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	if req.URL.RawQuery != "" {
		out = fmt.Sprintf("%s?%s", out, req.URL.RawQuery)
	}
	if len(out) > 65 {
		return fmt.Sprintf("%s...", out[:65])
	}
	return out
}

// statusClass groups a status code for styling: ok for 1xx to 3xx, warn for 4xx and error for 5xx
func statusClass(status int) string {
	switch {
	case status <= 0:
		return "none"
	case status >= 500:
		return "error"
	case status >= 400:
		return "warn"
	default:
		return "ok"
	}
}

func newHTMLTemplateModel(r *Recorder) (reportModel, error) {
	if len(r.Events) == 0 {
		return reportModel{}, errors.New("no events are defined")
	}

	namer := participantNamer(r.Meta)
	start := r.Events[0].GetTime()
	var events []reportEvent
	var participants []string
	seen := map[string]bool{}
	addParticipant := func(name string) {
		if !seen[name] {
			seen[name] = true
			participants = append(participants, name)
		}
	}

	for i, event := range r.Events {
		e := reportEvent{Number: i + 1}
		switch v := event.(type) {
		case HttpRequest:
			entry, err := newHTTPRequestLogEntry(v.Value)
			if err != nil {
				return reportModel{}, err
			}
			e.Kind, e.Source, e.Target = "request", v.Source, v.Target
			e.Label = formatDiagramRequest(v.Value)
			e.setWire(entry)
			e.Curl = curlCommand(v.Value, entry.RawBody)
		case HttpResponse:
			entry, err := newHTTPResponseLogEntry(v.Value)
			if err != nil {
				return reportModel{}, err
			}
			e.Kind, e.Source, e.Target = "response", v.Source, v.Target
			e.StatusCode = v.Value.StatusCode
			e.StatusClass = statusClass(e.StatusCode)
			e.Label = strings.TrimSpace(fmt.Sprintf("%d %s", v.Value.StatusCode, http.StatusText(v.Value.StatusCode)))
			e.setWire(entry)
		case MessageRequest:
			e.Kind, e.Source, e.Target = "request", v.Source, v.Target
			e.Label = v.Header
			e.setMessage(v.Header, v.Body)
		case MessageResponse:
			e.Kind, e.Source, e.Target = "response", v.Source, v.Target
			e.Label = v.Header
			e.setMessage(v.Header, v.Body)
		default:
			panic("received unknown event type")
		}
		e.Source, e.Target = namer.name(e.Source), namer.name(e.Target)
		e.Offset = formatOffset(event.GetTime().Sub(start))
		addParticipant(e.Source)
		addParticipant(e.Target)
		events = append(events, e)
	}

	markUnanswered(events)

	status, err := r.ResponseStatus()
	if err != nil {
		return reportModel{}, err
	}

	jsonMeta, err := json.Marshal(r.Meta)
	if err != nil {
		return reportModel{}, err
	}

	model := reportModel{
		Title:        r.Title,
		SubTitle:     r.SubTitle,
		StatusCode:   status,
		StatusText:   http.StatusText(status),
		StatusClass:  statusClass(status),
		Participants: participants,
		Events:       events,
		Diagram:      renderDiagram(participants, events),
		Meta:         metaItems(r.Meta),
		MetaJSON:     htmlTemplate.JS(jsonMeta),
	}
	if !start.IsZero() {
		model.StartedAt = start.UTC().Format("2006-01-02 15:04:05 MST")
	}
	if d, ok := r.Meta["duration"].(int64); ok {
		model.Duration = formatDuration(time.Duration(d))
	}
	return model, nil
}

// markUnanswered flags each request that is not followed by a response in the opposite direction.
// apitest records a call that matched no mock as a request without a response.
func markUnanswered(events []reportEvent) {
	pending := map[[2]string][]int{}
	for i, e := range events {
		key := [2]string{e.Source, e.Target}
		if e.Kind == "request" {
			pending[key] = append(pending[key], i)
			continue
		}
		reverse := [2]string{e.Target, e.Source}
		if open := pending[reverse]; len(open) > 0 {
			pending[reverse] = open[:len(open)-1]
		}
	}
	for _, open := range pending {
		for _, i := range open {
			events[i].Unanswered = true
		}
	}
}

// setWire fills the event from the wire representation of an HTTP message
func (e *reportEvent) setWire(entry logEntry) {
	lines := strings.Split(strings.TrimRight(entry.Header, "\r\n"), "\n")
	e.StartLine = strings.TrimSpace(lines[0])
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, "\r")
		if name, value, ok := strings.Cut(line, ":"); ok {
			e.Headers = append(e.Headers, headerLine{Name: name, Value: strings.TrimSpace(value)})
		}
	}
	e.setBody(entry.Body, entry.RawBody)
	e.Raw = strings.TrimRight(entry.Header, "\r\n")
	if entry.Body != "" {
		e.Raw += "\n\n" + entry.Body
	}
}

// setMessage fills the event from a custom message, such as a SQL query recorded by x/db
func (e *reportEvent) setMessage(header, body string) {
	e.StartLine = header
	e.setBody(formatMessageBody(body), body)
	e.Raw = header
	if body != "" {
		e.Raw += "\n\n" + body
	}
}

func (e *reportEvent) setBody(body, raw string) {
	e.Body = body
	if body == "" {
		return
	}
	e.BodySize = formatSize(len(raw))
	if html, ok := highlightJSON(body); ok {
		e.BodyKind = "json"
		e.BodyHTML = html
		return
	}
	e.BodyKind = "text"
	e.BodyHTML = htmlTemplate.HTML(htmlTemplate.HTMLEscapeString(body))
}

func newHTTPRequestLogEntry(req *http.Request) (logEntry, error) {
	// DumpRequest only fails while dumping the body, which is not requested here
	reqHeader, _ := httputil.DumpRequest(req, false)
	body, raw, err := formatBodyContent(req.Body, func(replacementBody io.ReadCloser) {
		req.Body = replacementBody
	})
	if err != nil {
		return logEntry{}, err
	}
	return logEntry{Header: string(reqHeader), Body: body, RawBody: raw}, nil
}

func newHTTPResponseLogEntry(res *http.Response) (logEntry, error) {
	// DumpResponse only fails while dumping the body, which is not requested here
	resDump, _ := httputil.DumpResponse(res, false)
	body, raw, err := formatBodyContent(res.Body, func(replacementBody io.ReadCloser) {
		res.Body = replacementBody
	})
	if err != nil {
		return logEntry{}, err
	}
	return logEntry{Header: string(resDump), Body: body, RawBody: raw}, nil
}

func metaItems(meta map[string]any) []metaItem {
	var items []metaItem
	for key, value := range meta {
		switch key {
		case "hash", "duration", "name", "path", "method", "status_code", "consumerName", "systemUnderTestName":
			// shown elsewhere in the report, or used to name the participants
			continue
		}
		items = append(items, metaItem{Key: key, Value: fmt.Sprint(value)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

func formatOffset(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return "+" + formatDuration(d)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return strconv.FormatFloat(float64(d)/float64(time.Microsecond), 'f', 0, 64) + " µs"
	case d < time.Second:
		return strconv.FormatFloat(float64(d)/float64(time.Millisecond), 'f', 1, 64) + " ms"
	default:
		return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + " s"
	}
}

func formatSize(n int) string {
	switch {
	case n < 1024:
		return strconv.Itoa(n) + " B"
	case n < 1024*1024:
		return strconv.FormatFloat(float64(n)/1024, 'f', 1, 64) + " KB"
	default:
		return strconv.FormatFloat(float64(n)/(1024*1024), 'f', 1, 64) + " MB"
	}
}
