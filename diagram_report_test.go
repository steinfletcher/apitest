package apitest

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestHighlightJSON_ClassifiesTokens(t *testing.T) {
	html, ok := highlightJSON(`{"name": "jon <3", "age": -1.5e2, "ok": true, "no": false, "x": null, "esc": "a\"b"}`)

	assert.True(t, ok)
	for _, want := range []string{
		`<span class="j-p">{</span>`,
		`<span class="j-k">&#34;name&#34;</span>`,
		`<span class="j-s">&#34;jon &lt;3&#34;</span>`,
		`<span class="j-n">-1.5e2</span>`,
		`<span class="j-b">true</span>`,
		`<span class="j-b">false</span>`,
		`<span class="j-z">null</span>`,
		`<span class="j-s">&#34;a\&#34;b&#34;</span>`,
	} {
		if !strings.Contains(string(html), want) {
			t.Errorf("highlighted JSON is missing %q in %s", want, html)
		}
	}
}

func TestHighlightJSON_RejectsOtherContent(t *testing.T) {
	for _, in := range []string{"", "   ", "SELECT * FROM users", `{"a":`} {
		_, ok := highlightJSON(in)
		assert.Equal(t, false, ok)
	}
}

func TestCurlCommand(t *testing.T) {
	get := httptest.NewRequest(http.MethodGet, "/user/1234?q=a", nil)
	get.Host = SystemUnderTestDefaultName
	get.Header.Set("Accept", "application/json")
	get.Header.Add("X-Tag", "a")
	get.Header.Add("X-Tag", "b")
	assert.Equal(t, "curl 'http://sut/user/1234?q=a' -H 'Accept: application/json' -H 'X-Tag: a' -H 'X-Tag: b'", curlCommand(get, ""))

	post, _ := http.NewRequest(http.MethodPost, "http://users/api/user", nil)
	assert.Equal(t, `curl -X POST 'http://users/api/user' --data-raw '{"name":"jo'\''n"}'`, curlCommand(post, `{"name":"jo'n"}`))

	noHost := &http.Request{Method: http.MethodDelete, URL: mustParseURL("/user/1"), Header: http.Header{}}
	assert.Equal(t, "curl -X DELETE 'http://sut/user/1'", curlCommand(noHost, ""))
}

func TestLayoutDiagram_FitsLabelsBetweenLifelines(t *testing.T) {
	long := strings.Repeat("x", 50)
	l := layoutDiagram([]string{"cli", "sut", "users"}, []reportEvent{
		{Number: 1, Source: "cli", Target: "sut", Label: "GET /"},
		{Number: 2, Source: "sut", Target: "users", Label: long},
		{Number: 3, Source: "users", Target: "users", Label: "self"},
	})

	if gap := l.x[2] - l.x[1]; gap < labelWidth(2, long) {
		t.Fatalf("expected the gap %v to fit the label width %v", gap, labelWidth(2, long))
	}
	if l.width < l.x[2]+selfLoopW+labelWidth(3, "self") {
		t.Fatalf("expected room for the self message to the right of the last participant, width %v", l.width)
	}
	assert.Equal(t, 2*diagramMarginY+2*participantH+firstRowOffset+3*rowH, l.height)
}

func TestRenderDiagram_DrawsSelfMessagesAsLoops(t *testing.T) {
	svg := string(renderDiagram([]string{"sut", "db"}, []reportEvent{
		{Number: 1, Kind: "request", Source: "sut", Target: "sut", Label: "cache hit"},
		{Number: 2, Kind: "request", Source: "db", Target: "sut", Label: strings.Repeat("y", 80)},
	}))

	if !regexp.MustCompile(`<path class="line" d="M[\d.]+ [\d.]+h30v16h-30"/>`).MatchString(svg) {
		t.Fatalf("expected a loop for the self message in %s", svg)
	}
	if !strings.Contains(svg, `text-anchor="start"`) || !strings.Contains(svg, "…") {
		t.Fatalf("expected a left aligned self label and a truncated long label")
	}
}

func TestReport_IsSelfContained(t *testing.T) {
	var out strings.Builder
	model, err := newHTMLTemplateModel(aDetailedRecorder())
	assert.NoError(t, err)

	assert.NoError(t, sequenceDiagramTemplate.Execute(&out, model))
	html := out.String()

	for _, external := range []string{`<script src`, `<link`, `@import`, `url(`, `src="http`, `href="//`} {
		if strings.Contains(html, external) {
			t.Errorf("report loads an external resource: found %q", external)
		}
	}
	urls := regexp.MustCompile(`https?://[^\s"'<>]+`).FindAllString(html, -1)
	for _, u := range urls {
		switch {
		case u == "http://www.w3.org/2000/svg",
			strings.HasPrefix(u, "https://apitest.dev"),
			strings.HasPrefix(u, "http://sut/"),
			strings.HasPrefix(u, "http://users/"):
		default:
			t.Errorf("unexpected URL in report: %s", u)
		}
	}
}

func TestReport_RendersEventsWithDetails(t *testing.T) {
	var out strings.Builder
	model, err := newHTMLTemplateModel(aDetailedRecorder())
	assert.NoError(t, err)

	assert.NoError(t, sequenceDiagramTemplate.Execute(&out, model))
	html := out.String()

	for _, want := range []string{
		`<title>GET /user/1234 · gets a user</title>`,
		`<span class="status ok">200 OK</span>`,
		`Duration <b>1.5 ms</b>`,
		`id="event-1"`, `id="event-4"`,
		`<dt>Authorization</dt><dd>Bearer s3cr3t</dd>`,
		`<span class="j-k">&#34;name&#34;</span>`,
		`Copy as curl`,
		`data-copy="curl &#39;http://users/api/user/1234&#39;`,
		`&#43;200 µs`,
		`<script type="application/json" id="metaJson">`,
		`env <b>ci</b>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("report is missing %q", want)
		}
	}
}

func TestReport_FormattingHelpers(t *testing.T) {
	assert.Equal(t, "450 µs", formatDuration(450*time.Microsecond))
	assert.Equal(t, "12.3 ms", formatDuration(12340*time.Microsecond))
	assert.Equal(t, "2.50 s", formatDuration(2500*time.Millisecond))
	assert.Equal(t, "+0 µs", formatOffset(-time.Second))
	assert.Equal(t, "512 B", formatSize(512))
	assert.Equal(t, "1.5 KB", formatSize(1536))
	assert.Equal(t, "2.0 MB", formatSize(2*1024*1024))
	assert.Equal(t, "ab…", truncate("abcdef", 3))
	assert.Equal(t, "abc", truncate("abc", 3))
	assert.Equal(t, "?", displayName(""))
	assert.Equal(t, "12.3", num(12.345))
	assert.Equal(t, "{\n  \"a\": 1\n}", formatMessageBody(`{"a":1}`))
	assert.Equal(t, "SELECT 1", formatMessageBody("SELECT 1"))
}

func TestReport_MessageEventsHighlightJSONBodies(t *testing.T) {
	recorder := NewTestRecorder().
		AddMessageRequest(MessageRequest{Source: "sut", Target: "kafka", Header: "publish orders", Body: `{"id":1}`}).
		AddMessageResponse(MessageResponse{Source: "kafka", Target: "sut", Header: "ack"})

	model, err := newHTMLTemplateModel(recorder)

	assert.NoError(t, err)
	assert.Equal(t, -1, model.StatusCode)
	assert.Equal(t, "none", model.StatusClass)
	assert.Equal(t, "json", model.Events[0].BodyKind)
	assert.Equal(t, "8 B", model.Events[0].BodySize)
	assert.Equal(t, "publish orders\n\n{\"id\":1}", model.Events[0].Raw)
	assert.Equal(t, "", model.Events[1].Body)
	assert.Equal(t, "ack", model.Events[1].Raw)
	assert.Equal(t, "", model.StartedAt)
}

func aDetailedRecorder() *Recorder {
	start := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	inbound := httptest.NewRequest(http.MethodGet, "/user/1234", nil)
	inbound.Host = SystemUnderTestDefaultName
	outbound, _ := http.NewRequest(http.MethodGet, "http://users/api/user/1234", nil)
	outbound.Header.Set("Authorization", "Bearer s3cr3t")

	return NewTestRecorder().
		AddTitle("GET /user/1234").
		AddSubTitle("gets a user").
		AddMeta(map[string]any{"hash": "abc", "duration": int64(1500 * time.Microsecond), "env": "ci"}).
		AddHttpRequest(HttpRequest{Source: ConsumerDefaultName, Target: SystemUnderTestDefaultName, Value: inbound, Timestamp: start}).
		AddHttpRequest(HttpRequest{Source: SystemUnderTestDefaultName, Target: "users", Value: outbound, Timestamp: start.Add(200 * time.Microsecond)}).
		AddHttpResponse(HttpResponse{Source: "users", Target: SystemUnderTestDefaultName, Timestamp: start.Add(900 * time.Microsecond), Value: &http.Response{
			StatusCode: http.StatusOK, ProtoMajor: 1, ProtoMinor: 1,
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   http.NoBody,
		}}).
		AddHttpResponse(HttpResponse{Source: SystemUnderTestDefaultName, Target: ConsumerDefaultName, Timestamp: start.Add(1500 * time.Microsecond), Value: &http.Response{
			StatusCode: http.StatusOK, ProtoMajor: 1, ProtoMinor: 1,
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   ioNopCloser(`{"name": "jon", "id": "1234"}`),
		}})
}

func ioNopCloser(s string) *nopReadCloser { return &nopReadCloser{strings.NewReader(s)} }

type nopReadCloser struct{ *strings.Reader }

func (nopReadCloser) Close() error { return nil }

func TestNewHTMLTemplateModel_FlagsRequestsWithoutAResponse(t *testing.T) {
	recorder := NewTestRecorder().
		AddHttpRequest(HttpRequest{Source: ConsumerDefaultName, Target: SystemUnderTestDefaultName, Value: httptest.NewRequest(http.MethodGet, "/", nil)}).
		AddHttpRequest(HttpRequest{Source: SystemUnderTestDefaultName, Target: "billing", Value: httptest.NewRequest(http.MethodGet, "/invoices", nil)}).
		AddHttpRequest(HttpRequest{Source: SystemUnderTestDefaultName, Target: "catalog", Value: httptest.NewRequest(http.MethodGet, "/products", nil)}).
		AddHttpResponse(HttpResponse{Source: "catalog", Target: SystemUnderTestDefaultName, Value: &http.Response{StatusCode: http.StatusOK}}).
		AddHttpResponse(HttpResponse{Source: SystemUnderTestDefaultName, Target: ConsumerDefaultName, Value: &http.Response{StatusCode: http.StatusBadGateway}})

	model, err := newHTMLTemplateModel(recorder)

	assert.NoError(t, err)
	var unanswered []int
	for _, e := range model.Events {
		if e.Unanswered {
			unanswered = append(unanswered, e.Number)
		}
	}
	assert.Equal(t, []int{2}, unanswered)
	if !strings.Contains(string(model.Diagram), `class="msg request unanswered"`) || !strings.Contains(string(model.Diagram), `<g class="no-reply">`) {
		t.Fatal("expected the unanswered request to be marked in the diagram")
	}
}
