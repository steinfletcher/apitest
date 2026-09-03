package apitest

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

// SystemUnderTestDefaultName default name for system under test
const SystemUnderTestDefaultName = "sut"

// ConsumerDefaultName default consumer name
const ConsumerDefaultName = "cli"

// APITest is the top level struct holding the test spec
type APITest struct {
	debugEnabled             bool
	mockResponseDelayEnabled bool
	networkingEnabled        bool
	networkingHTTPClient     *http.Client
	reporter                 ReportFormatter
	verifier                 Verifier
	recorder                 *Recorder
	handler                  http.Handler
	name                     string
	host                     string
	request                  *Request
	response                 *Response
	observers                []Observe
	mocksObservers           []Observe
	recorderHook             RecorderHook
	mocks                    []*Mock
	t                        TestingT
	httpClient               *http.Client
	httpRequest              *http.Request
	transport                *Transport
	meta                     map[string]any
	started                  time.Time
	finished                 time.Time
	fileSystem               fs.FS
}

// InboundRequest used to wrap the incoming request with a timestamp
//
// Deprecated: InboundRequest is not used by apitest and is retained only for API compatibility.
type InboundRequest struct {
	request   *http.Request
	timestamp time.Time
}

// FinalResponse used to wrap the final response with a timestamp
//
// Deprecated: FinalResponse is not used by apitest and is retained only for API compatibility.
type FinalResponse struct {
	response  *http.Response
	timestamp time.Time
}

// Observe will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request, *APITest)

// RecorderHook used to implement a custom interaction recorder
type RecorderHook func(*Recorder)

// New creates a new api test. The name is optional and will appear in test reports
func New(name ...string) *APITest {
	apiTest := &APITest{
		meta: map[string]any{},
	}

	request := &Request{
		apiTest:  apiTest,
		headers:  map[string][]string{},
		query:    map[string][]string{},
		formData: map[string][]string{},
	}
	response := &Response{
		apiTest: apiTest,
		headers: map[string][]string{},
	}
	apiTest.request = request
	apiTest.response = response

	if len(name) > 0 {
		apiTest.name = name[0]
	}
	apiTest.fileSystem = OSFS{}

	return apiTest
}

// Handler is a convenience method for creating a new apitest with a handler
func Handler(handler http.Handler) *APITest {
	return New().Handler(handler)
}

// HandlerFunc is a convenience method for creating a new apitest with a handler func
func HandlerFunc(handlerFunc http.HandlerFunc) *APITest {
	return New().HandlerFunc(handlerFunc)
}

// EnableNetworking will enable networking for provided clients
func (a *APITest) EnableNetworking(cli ...*http.Client) *APITest {
	a.networkingEnabled = true
	if len(cli) == 1 {
		a.networkingHTTPClient = cli[0]
		return a
	}
	a.networkingHTTPClient = http.DefaultClient
	return a
}

// EnableMockResponseDelay turns on mock response delays (defaults to OFF)
func (a *APITest) EnableMockResponseDelay() *APITest {
	a.mockResponseDelayEnabled = true
	return a
}

// Debug logs to the console the http wire representation of all http interactions that are intercepted by apitest. This includes the inbound request to the application under test, the response returned by the application and any interactions that are intercepted by the mock server.
func (a *APITest) Debug() *APITest {
	a.debugEnabled = true
	return a
}

// Report provides a hook to add custom formatting to the output of the test
func (a *APITest) Report(reporter ReportFormatter) *APITest {
	a.reporter = reporter
	return a
}

// Recorder provides a hook to add a recorder to the test
func (a *APITest) Recorder(recorder *Recorder) *APITest {
	a.recorder = recorder
	return a
}

// Meta provides a hook to add custom meta data to the test which can be picked up when defining a custom reporter
func (a *APITest) Meta(meta map[string]any) *APITest {
	a.meta = meta
	return a
}

// Handler defines the http handler that is invoked when the test is run
func (a *APITest) Handler(handler http.Handler) *APITest {
	a.handler = handler
	return a
}

// HandlerFunc defines the http handler that is invoked when the test is run
func (a *APITest) HandlerFunc(handlerFunc http.HandlerFunc) *APITest {
	a.handler = handlerFunc
	return a
}

// Mocks is a builder method for setting the mocks
func (a *APITest) Mocks(mocks ...*Mock) *APITest {
	var m []*Mock
	for i := range mocks {
		if mocks[i].anyTimesSet {
			m = append(m, mocks[i])
			continue
		}

		times := mocks[i].response.mock.times
		for j := 1; j <= times; j++ {
			mockCpy := mocks[i].copy()
			mockCpy.times = 1
			m = append(m, mockCpy)
		}
	}
	a.mocks = m
	return a
}

// HttpClient allows the developer to provide a custom http client when using mocks
func (a *APITest) HttpClient(cli *http.Client) *APITest {
	a.httpClient = cli
	return a
}

// Observe is a builder method for setting the observers
func (a *APITest) Observe(observers ...Observe) *APITest {
	a.observers = observers
	return a
}

// ObserveMocks is a builder method for setting the mocks observers
func (a *APITest) ObserveMocks(observer Observe) *APITest {
	a.mocksObservers = append(a.mocksObservers, observer)
	return a
}

// RecorderHook allows the consumer to provider a function that will receive the recorder instance before the
// test runs. This can be used to inject custom events which can then be rendered in diagrams
// Deprecated: use Recorder() instead
func (a *APITest) RecorderHook(hook RecorderHook) *APITest {
	a.recorderHook = hook
	return a
}

// Request returns the request spec
func (a *APITest) Request() *Request {
	return a.request
}

// Response returns the expected response
func (a *APITest) Response() *Response {
	return a.response
}

// Use filesystem allows you to change to a custom fs.FS filesystem that you can define (Used by Request.MultipartFile).
// e.g: fstest.MapFS
// Your os filesystem is used by default
func (a *APITest) UseFS(fs fs.FS) *APITest {
	a.fileSystem = fs
	return a
}

// Host is a builder method for explicitly setting the host
func (a *APITest) Host(host string) *APITest {
	a.host = host
	return a
}

// Intercept is a builder method for setting the request interceptor
func (a *APITest) Intercept(interceptor Intercept) *APITest {
	a.request.interceptor = interceptor
	return a
}

// Verifier allows consumers to override the verification implementation.
func (a *APITest) Verifier(v Verifier) *APITest {
	a.verifier = v
	return a
}

// Method is a builder method for setting the http method of the request
func (a *APITest) Method(method string) *Request {
	a.request.method = method
	return a.request
}

// HttpRequest defines the native `http.Request`
func (a *APITest) HttpRequest(req *http.Request) *Request {
	a.httpRequest = req
	return a.request
}

// Get is a convenience method for setting the request as http.MethodGet
func (a *APITest) Get(url string) *Request {
	a.request.method = http.MethodGet
	a.request.url = url
	return a.request
}

// Getf is a convenience method that adds formatting support to Get
func (a *APITest) Getf(format string, args ...any) *Request {
	return a.Get(fmt.Sprintf(format, args...))
}

// Post is a convenience method for setting the request as http.MethodPost
func (a *APITest) Post(url string) *Request {
	a.request.method = http.MethodPost
	a.request.url = url
	return a.request
}

// Postf is a convenience method that adds formatting support to Post
func (a *APITest) Postf(format string, args ...any) *Request {
	return a.Post(fmt.Sprintf(format, args...))
}

// Put is a convenience method for setting the request as http.MethodPut
func (a *APITest) Put(url string) *Request {
	a.request.method = http.MethodPut
	a.request.url = url
	return a.request
}

// Putf is a convenience method that adds formatting support to Put
func (a *APITest) Putf(format string, args ...any) *Request {
	return a.Put(fmt.Sprintf(format, args...))
}

// Delete is a convenience method for setting the request as http.MethodDelete
func (a *APITest) Delete(url string) *Request {
	a.request.method = http.MethodDelete
	a.request.url = url
	return a.request
}

// Deletef is a convenience method that adds formatting support to Delete
func (a *APITest) Deletef(format string, args ...any) *Request {
	return a.Delete(fmt.Sprintf(format, args...))
}

// Patch is a convenience method for setting the request as http.MethodPatch
func (a *APITest) Patch(url string) *Request {
	a.request.method = http.MethodPatch
	a.request.url = url
	return a.request
}

// Patchf is a convenience method that adds formatting support to Patch
func (a *APITest) Patchf(format string, args ...any) *Request {
	return a.Patch(fmt.Sprintf(format, args...))
}

// End runs the test returning the result to the caller
func (r *Response) End() Result {
	apiTest := r.apiTest
	defer func() {
		if apiTest.debugEnabled {
			apiTest.finished = time.Now()
			fmt.Printf("Duration: %s\n", apiTest.finished.Sub(apiTest.started))
		}
	}()

	if apiTest.handler == nil && !apiTest.networkingEnabled {
		apiTest.t.Fatal("either define a http.Handler or enable networking")
	}

	apiTest.started = time.Now()
	var res *http.Response
	if apiTest.reporter != nil {
		res = apiTest.report()
	} else {
		res = r.runTest()
	}

	var unmatchedMocks []UnmatchedMock
	for _, m := range r.apiTest.mocks {
		if !m.isUsed {
			unmatchedMocks = append(unmatchedMocks, UnmatchedMock{
				URL: *m.request.url,
			})
		}
	}

	return Result{
		Response:       res,
		unmatchedMocks: unmatchedMocks,
	}
}

type mockInteraction struct {
	request   *http.Request
	response  *http.Response
	timestamp time.Time
}

func (r *mockInteraction) GetRequestHost() string {
	host := r.request.Host
	if host == "" {
		host = r.request.URL.Host
	}
	return host
}

func (a *APITest) report() *http.Response {
	var capturedInboundReq *http.Request
	var capturedFinalRes *http.Response
	var capturedMockInteractions []*mockInteraction
	var capturedMockInteractionsMu sync.Mutex // mocks may be invoked concurrently by the handler

	a.observers = append(a.observers, func(finalRes *http.Response, inboundReq *http.Request, a *APITest) {
		capturedFinalRes = copyHttpResponse(finalRes)
		capturedInboundReq = copyHttpRequest(inboundReq)
	})

	a.mocksObservers = append(a.mocksObservers, func(mockRes *http.Response, mockReq *http.Request, a *APITest) {
		interaction := &mockInteraction{
			request:   copyHttpRequest(mockReq),
			response:  copyHttpResponse(mockRes),
			timestamp: time.Now().UTC(),
		}
		capturedMockInteractionsMu.Lock()
		capturedMockInteractions = append(capturedMockInteractions, interaction)
		capturedMockInteractionsMu.Unlock()
	})

	if a.recorder == nil {
		a.recorder = NewTestRecorder()
	}
	defer a.recorder.Reset()

	if a.recorderHook != nil {
		a.recorderHook(a.recorder)
	}

	a.started = time.Now()
	res := a.response.runTest()
	a.finished = time.Now()

	a.recorder.
		AddTitle(fmt.Sprintf("%s %s", capturedInboundReq.Method, capturedInboundReq.URL.String())).
		AddSubTitle(a.name).
		AddHttpRequest(HttpRequest{
			Source:    ConsumerDefaultName,
			Target:    SystemUnderTestDefaultName,
			Value:     capturedInboundReq,
			Timestamp: a.started,
		})

	for _, interaction := range capturedMockInteractions {
		a.recorder.AddHttpRequest(HttpRequest{
			Source:    SystemUnderTestDefaultName,
			Target:    interaction.GetRequestHost(),
			Value:     interaction.request,
			Timestamp: interaction.timestamp,
		})
		if interaction.response != nil {
			a.recorder.AddHttpResponse(HttpResponse{
				Source:    interaction.GetRequestHost(),
				Target:    SystemUnderTestDefaultName,
				Value:     interaction.response,
				Timestamp: interaction.timestamp,
			})
		}
	}

	a.recorder.AddHttpResponse(HttpResponse{
		Source:    SystemUnderTestDefaultName,
		Target:    ConsumerDefaultName,
		Value:     capturedFinalRes,
		Timestamp: a.finished,
	})

	sort.Slice(a.recorder.Events, func(i, j int) bool {
		return a.recorder.Events[i].GetTime().Before(a.recorder.Events[j].GetTime())
	})

	meta := map[string]any{}

	maps.Copy(meta, a.meta)

	meta["status_code"] = capturedFinalRes.StatusCode
	meta["path"] = capturedInboundReq.URL.String()
	meta["method"] = capturedInboundReq.Method
	meta["name"] = a.name
	meta["hash"] = createHash(meta)
	meta["duration"] = a.finished.Sub(a.started).Nanoseconds()

	a.recorder.AddMeta(meta)
	a.reporter.Format(a.recorder)

	return res
}

func createHash(meta map[string]any) string {
	path := meta["path"]
	method := meta["method"]
	name := meta["name"]
	app := meta["app"]

	// hash.Hash.Write never returns an error
	prefix := fnv.New32a()
	_, _ = prefix.Write(fmt.Appendf(nil, "%s%s%s", app, strings.ToUpper(method.(string)), path))

	suffix := fnv.New32a()
	_, _ = suffix.Write([]byte(name.(string)))
	return fmt.Sprintf("%d_%d", prefix.Sum32(), suffix.Sum32())
}

func (r *Response) runTest() *http.Response {
	a := r.apiTest
	if len(a.mocks) > 0 {
		a.transport = newTransport(
			a.mocks,
			a.httpClient,
			a.debugEnabled,
			a.mockResponseDelayEnabled,
			a.mocksObservers,
			r.apiTest,
		)
		defer a.transport.Reset()
		a.transport.Hijack()
	}
	res, req := a.doRequest()

	defer func() {
		if len(a.observers) > 0 {
			for _, observe := range a.observers {
				observe(res, req, a)
			}
		}
	}()

	if a.verifier == nil {
		a.verifier = DefaultVerifier{}
	}

	a.assertMocks()
	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)
	a.assertFunc(res, req)

	return copyHttpResponse(res)
}

func (a *APITest) doRequest() (*http.Response, *http.Request) {
	req := a.buildRequest()
	if a.request.interceptor != nil {
		a.request.interceptor(req)
	}
	resRecorder := httptest.NewRecorder()

	if a.debugEnabled {
		requestDump, err := httputil.DumpRequest(req, true)
		if err == nil {
			debugLog(requestDebugPrefix, "inbound http request", string(requestDump))
		}
	}

	var res *http.Response
	var err error
	if !a.networkingEnabled {
		a.serveHttp(resRecorder, copyHttpRequest(req))
		res = resRecorder.Result()
	} else {
		res, err = a.networkingHTTPClient.Do(copyHttpRequest(req))
		if err != nil {
			a.t.Fatal(err)
		}
	}

	if a.debugEnabled {
		responseDump, err := httputil.DumpResponse(res, true)
		if err == nil {
			debugLog(responseDebugPrefix, "final response", string(responseDump))
		}
	}

	return res, req
}

func (a *APITest) serveHttp(res *httptest.ResponseRecorder, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			a.t.Fatalf("%s: %s", err, debug.Stack())
		}
	}()

	a.handler.ServeHTTP(res, req)
}
