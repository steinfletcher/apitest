package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const systemUnderTestDefaultName = "sut"
const consumerName = "cli"

var divider = strings.Repeat("-", 10)
var requestDebugPrefix = fmt.Sprintf("%s>", divider)
var responseDebugPrefix = fmt.Sprintf("<%s", divider)

// APITest is the top level struct holding the test spec
type APITest struct {
	debugEnabled  bool
	reporter      ReportFormatter
	recorder      *Recorder
	handler       http.Handler
	name          string
	request       *Request
	response      *Response
	observers     []Observe
	mocksObserver Observe
	recorderHook  RecorderHook
	mocks         []*Mock
	t             *testing.T
	httpClient    *http.Client
	transport     *Transport
	meta          map[string]interface{}
}

type InboundRequest struct {
	request   *http.Request
	timestamp time.Time
}

type FinalResponse struct {
	response  *http.Response
	timestamp time.Time
}

// Observe will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request, *APITest)

type RecorderHook func(*Recorder)

// New creates a new api test. The name is optional and will appear in test reports
func New(name ...string) *APITest {
	apiTest := &APITest{
		meta: map[string]interface{}{},
	}

	request := &Request{
		apiTest: apiTest,
		headers: map[string][]string{},
		query:   map[string][]string{},
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

	return apiTest
}

// Debug logs to the console the http wire representation of all http interactions that are intercepted by apitest. This includes the inbound request to the application under test, the response returned by the application and any interactions that are intercepted by the mock server.
func (a *APITest) Debug() *APITest {
	a.debugEnabled = true
	return a
}

// Reporter provides a hook to add custom formatting to the output of the test
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
func (a *APITest) Meta(meta map[string]interface{}) *APITest {
	a.meta = meta
	return a
}

// Handler defines the http handler that is invoked when the test is run
func (a *APITest) Handler(handler http.Handler) *APITest {
	a.handler = handler
	return a
}

// Mocks is a builder method for setting the mocks
func (a *APITest) Mocks(mocks ...*Mock) *APITest {
	var m []*Mock
	for i := range mocks {
		times := mocks[i].response.times
		for j := 1; j <= times; j++ {
			mockCopy := *mocks[i]
			m = append(m, &mockCopy)
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
	a.mocksObserver = observer
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

// Request is the user defined request that will be invoked on the handler under test
type Request struct {
	interceptor     Intercept
	method          string
	url             string
	body            string
	query           map[string][]string
	queryCollection map[string][]string
	headers         map[string][]string
	cookies         []*Cookie
	basicAuth       string
	apiTest         *APITest
}

// Intercept will be called before the request is made. Updates to the request will be reflected in the test
type Intercept func(*http.Request)

type pair struct {
	l string
	r string
}

// Intercept is a builder method for setting the request interceptor
func (a *APITest) Intercept(interceptor Intercept) *APITest {
	a.request.interceptor = interceptor
	return a
}

// Method is a builder method for setting the http method of the request
func (a *APITest) Method(method string) *Request {
	a.request.method = method
	return a.request
}

// Get is a convenience method for setting the request as http.MethodGet
func (a *APITest) Get(url string) *Request {
	a.request.method = http.MethodGet
	a.request.url = url
	return a.request
}

// Post is a convenience method for setting the request as http.MethodPost
func (a *APITest) Post(url string) *Request {
	r := a.request
	r.method = http.MethodPost
	r.url = url
	return r
}

// Put is a convenience method for setting the request as http.MethodPut
func (a *APITest) Put(url string) *Request {
	r := a.request
	r.method = http.MethodPut
	r.url = url
	return r
}

// Delete is a convenience method for setting the request as http.MethodDelete
func (a *APITest) Delete(url string) *Request {
	a.request.method = http.MethodDelete
	a.request.url = url
	return a.request
}

// Patch is a convenience method for setting the request as http.MethodPatch
func (a *APITest) Patch(url string) *Request {
	a.request.method = http.MethodPatch
	a.request.url = url
	return a.request
}

// URL is a builder method for setting the url of the request
func (r *Request) URL(url string) *Request {
	r.url = url
	return r
}

// Body is a builder method to set the request body
func (r *Request) Body(b string) *Request {
	r.body = b
	return r
}

// JSON is a convenience method for setting the request body and content type header as "application/json"
func (r *Request) JSON(b string) *Request {
	r.body = b
	r.Header("Content-Type", "application/json")
	return r
}

// Query is a convenience method to add a query parameter to the request.
func (r *Request) Query(key, value string) *Request {
	r.query[key] = append(r.query[key], value)
	return r
}

// QueryParams is a builder method to set the request query parameters.
// This can be used in combination with request.QueryCollection
func (r *Request) QueryParams(params map[string]string) *Request {
	for k, v := range params {
		r.query[k] = append(r.query[k], v)
	}
	return r
}

// QueryCollection is a builder method to set the request query parameters
// This can be used in combination with request.Query
func (r *Request) QueryCollection(q map[string][]string) *Request {
	r.queryCollection = q
	return r
}

// Header is a builder method to set the request headers
func (r *Request) Header(key, value string) *Request {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers is a builder method to set the request headers
func (r *Request) Headers(headers map[string]string) *Request {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

// Cookie is a convenience method for setting a single request cookies by name and value
func (r *Request) Cookie(name, value string) *Request {
	r.cookies = append(r.cookies, &Cookie{name: &name, value: &value})
	return r
}

// Cookies is a builder method to set the request cookies
func (r *Request) Cookies(c ...*Cookie) *Request {
	r.cookies = append(r.cookies, c...)
	return r
}

// BasicAuth is a builder method to sets basic auth on the request.
func (r *Request) BasicAuth(username, password string) *Request {
	r.basicAuth = fmt.Sprintf("%s:%s", username, password)
	return r
}

// Expect marks the request spec as complete and following code will define the expected response
func (r *Request) Expect(t *testing.T) *Response {
	r.apiTest.t = t
	return r.apiTest.response
}

// Response is the user defined expected response from the application under test
type Response struct {
	status             int
	body               string
	headers            map[string][]string
	cookies            []*Cookie
	cookiesPresent     []string
	cookiesNotPresent  []string
	jsonPathExpression string
	jsonPathAssert     func(interface{})
	apiTest            *APITest
	assert             []Assert
}

// Assert is a user defined custom assertion function
type Assert func(*http.Response, *http.Request) error

// Body is the expected response body
func (r *Response) Body(b string) *Response {
	r.body = b
	return r
}

// Cookies is the expected response cookies
func (r *Response) Cookies(cookies ...*Cookie) *Response {
	r.cookies = append(r.cookies, cookies...)
	return r
}

// Cookie is used to match on an individual cookie name/value pair in the expected response cookies
func (r *Response) Cookie(name, value string) *Response {
	r.cookies = append(r.cookies, NewCookie(name).Value(value))
	return r
}

// CookiePresent is used to assert that a cookie is present in the response,
// regardless of its value
func (r *Response) CookiePresent(cookieName string) *Response {
	r.cookiesPresent = append(r.cookiesPresent, cookieName)
	return r
}

// CookieNotPresent is used to assert that a cookie is not present in the response
func (r *Response) CookieNotPresent(cookieName string) *Response {
	r.cookiesNotPresent = append(r.cookiesNotPresent, cookieName)
	return r
}

// Header is a builder method to set the request headers
func (r *Response) Header(key, value string) *Response {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers is a builder method to set the request headers
func (r *Response) Headers(headers map[string]string) *Response {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[textproto.CanonicalMIMEHeaderKey(normalizedKey)], v)
	}
	return r
}

// Status is the expected response http status code
func (r *Response) Status(s int) *Response {
	r.status = s
	return r
}

// Assert allows the consumer to provide a user defined function containing their own
// custom assertions
func (r *Response) Assert(fn func(*http.Response, *http.Request) error) *Response {
	r.assert = append(r.assert, fn)
	return r.apiTest.response
}

// End runs the test and all defined assertions
func (r *Response) End() {
	if r.apiTest.reporter != nil {
		r.apiTest.report()
		return
	}
	r.execute()
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

func (a *APITest) report() {
	var capturedInboundReq *http.Request
	var capturedFinalRes *http.Response
	var capturedMockInteractions []*mockInteraction

	a.observers = []Observe{
		func(finalRes *http.Response, inboundReq *http.Request, a *APITest) {
			capturedFinalRes = finalRes
			capturedInboundReq = inboundReq
		},
	}
	a.mocksObserver = func(mockRes *http.Response, mockReq *http.Request, a *APITest) {
		capturedMockInteractions = append(capturedMockInteractions, &mockInteraction{
			request:   copyHttpRequest(mockReq),
			response:  copyHttpResponse(mockRes),
			timestamp: time.Now().UTC(),
		})
	}

	if a.recorder == nil {
		a.recorder = NewTestRecorder()
	}

	if a.recorderHook != nil {
		a.recorderHook(a.recorder)
	}

	execTime := time.Now().UTC()
	a.response.execute()
	finishTime := time.Now().UTC()

	a.recorder.
		AddTitle(fmt.Sprintf("%s %s", capturedInboundReq.Method, capturedInboundReq.URL.String())).
		AddSubTitle(a.name).
		AddHttpRequest(HttpRequest{
			Source:    quoted(consumerName),
			Target:    quoted(systemUnderTestDefaultName),
			Value:     capturedInboundReq,
			Timestamp: execTime,
		})

	for _, interaction := range capturedMockInteractions {
		a.recorder.AddHttpRequest(HttpRequest{
			Source:    quoted(systemUnderTestDefaultName),
			Target:    quoted(interaction.GetRequestHost()),
			Value:     interaction.request,
			Timestamp: interaction.timestamp,
		})
		if interaction.response != nil {
			a.recorder.AddHttpResponse(HttpResponse{
				Source:    quoted(interaction.GetRequestHost()),
				Target:    quoted(systemUnderTestDefaultName),
				Value:     interaction.response,
				Timestamp: interaction.timestamp,
			})
		}
	}

	a.recorder.AddHttpResponse(HttpResponse{
		Source:    quoted(systemUnderTestDefaultName),
		Target:    quoted(consumerName),
		Value:     capturedFinalRes,
		Timestamp: finishTime,
	})

	sort.Slice(a.recorder.Events, func(i, j int) bool {
		return a.recorder.Events[i].GetTime().Before(a.recorder.Events[j].GetTime())
	})

	meta := map[string]interface{}{}

	for k, v := range a.meta {
		meta[k] = v
	}

	meta["status_code"] = capturedFinalRes.StatusCode
	meta["path"] = capturedInboundReq.URL.String()
	meta["method"] = capturedInboundReq.Method
	meta["name"] = a.name
	meta["hash"] = createHash(meta)

	a.recorder.AddMeta(meta)

	a.reporter.Format(a.recorder)
}

func createHash(meta map[string]interface{}) string {
	path := meta["path"]
	method := meta["method"]
	name := meta["name"]
	app := meta["app"]

	prefix := fnv.New32a()
	_, err := prefix.Write([]byte(fmt.Sprintf("%s%s%s", app, strings.ToUpper(method.(string)), path)))
	if err != nil {
		panic(err)
	}

	suffix := fnv.New32a()
	_, err = suffix.Write([]byte(name.(string)))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%d_%d", prefix.Sum32(), suffix.Sum32())
}

func (r *Response) execute() {
	apiTest := r.apiTest
	if len(apiTest.mocks) > 0 {
		apiTest.transport = newTransport(
			apiTest.mocks,
			apiTest.httpClient,
			apiTest.debugEnabled,
			apiTest.mocksObserver,
			r.apiTest,
		)
		defer apiTest.transport.Reset()
		apiTest.transport.Hijack()
	}
	apiTest.run()
}

func (a *APITest) run() {
	res, req := a.runTest()

	defer func() {
		if len(a.observers) > 0 {
			for _, observe := range a.observers {
				observe(res.Result(), req, a)
			}
		}
	}()

	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)
	err := a.assertFunc(res, req)
	if err != nil {
		a.t.Fatal(err.Error())
	}
}

func (a *APITest) assertFunc(res *httptest.ResponseRecorder, req *http.Request) error {
	if len(a.response.assert) > 0 {
		for _, assertFn := range a.response.assert {
			response := res.Result()
			err := assertFn(copyHttpResponse(response), req)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *APITest) runTest() (*httptest.ResponseRecorder, *http.Request) {
	req := a.BuildRequest()
	if a.request.interceptor != nil {
		a.request.interceptor(req)
	}
	res := httptest.NewRecorder()

	if a.debugEnabled {
		requestDump, err := httputil.DumpRequest(req, true)
		if err == nil {
			debugLog(requestDebugPrefix, "inbound http request", string(requestDump))
		}
	}

	a.serveHttp(res, req)

	if a.debugEnabled {
		responseDump, err := httputil.DumpResponse(res.Result(), true)
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

func (a *APITest) BuildRequest() *http.Request {
	req, _ := http.NewRequest(a.request.method, a.request.url, bytes.NewBufferString(a.request.body))
	req.URL.RawQuery = formatQuery(a.request)
	req.Host = "application"

	for k, v := range a.request.headers {
		for _, headerValue := range v {
			req.Header.Add(k, headerValue)
		}
	}

	for _, cookie := range a.request.cookies {
		req.AddCookie(cookie.ToHttpCookie())
	}

	if a.request.basicAuth != "" {
		parts := strings.Split(a.request.basicAuth, ":")
		req.SetBasicAuth(parts[0], parts[1])
	}

	return req
}

func formatQuery(request *Request) string {
	var out url.Values = map[string][]string{}

	if request.queryCollection != nil {
		for _, param := range buildQueryCollection(request.queryCollection) {
			out.Add(param.l, param.r)
		}
	}

	if request.query != nil {
		for k, v := range request.query {
			for _, p := range v {
				out.Add(k, p)
			}
		}
	}

	if len(out) > 0 {
		return out.Encode()
	}

	return ""
}

func buildQueryCollection(params map[string][]string) []pair {
	if len(params) == 0 {
		return []pair{}
	}

	var pairs []pair
	for k, v := range params {
		for _, paramValue := range v {
			pairs = append(pairs, pair{l: k, r: paramValue})
		}
	}
	return pairs
}

func (a *APITest) assertResponse(res *httptest.ResponseRecorder) {
	if a.response.status != 0 {
		assert.Equal(a.t, a.response.status, res.Code, fmt.Sprintf("Status code %d not equal to %d", res.Code, a.response.status))
	}

	if a.response.body != "" {
		if json.Valid([]byte(a.response.body)) {
			assert.JSONEq(a.t, a.response.body, res.Body.String())
		} else {
			assert.Equal(a.t, a.response.body, res.Body.String())
		}
	}
}

func (a *APITest) assertCookies(response *httptest.ResponseRecorder) {
	if len(a.response.cookies) > 0 {
		for _, expectedCookie := range a.response.cookies {
			var mismatchedFields []string
			foundCookie := false
			for _, actualCookie := range responseCookies(response) {
				cookieFound, errors := compareCookies(expectedCookie, actualCookie)
				if cookieFound {
					foundCookie = true
					mismatchedFields = append(mismatchedFields, errors...)
				}
			}
			assert.Equal(a.t, true, foundCookie, "ExpectedCookie not found - "+*expectedCookie.name)
			assert.Equal(a.t, 0, len(mismatchedFields), strings.Join(mismatchedFields, ","))
		}
	}

	if len(a.response.cookiesPresent) > 0 {
		for _, cookieName := range a.response.cookiesPresent {
			foundCookie := false
			for _, cookie := range responseCookies(response) {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			assert.Equal(a.t, true, foundCookie, "ExpectedCookie not found - "+cookieName)
		}
	}

	if len(a.response.cookiesNotPresent) > 0 {
		for _, cookieName := range a.response.cookiesNotPresent {
			foundCookie := false
			for _, cookie := range responseCookies(response) {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			assert.Equal(a.t, false, foundCookie, "ExpectedCookie found - "+cookieName)
		}
	}
}

func responseCookies(response *httptest.ResponseRecorder) []*http.Cookie {
	return response.Result().Cookies()
}

func (a *APITest) assertHeaders(res *httptest.ResponseRecorder) {
	for expectedHeader, expectedValues := range a.response.headers {
		for _, expectedValue := range expectedValues {
			found := false
			result := res.Result()
			for _, resValue := range result.Header[expectedHeader] {
				if expectedValue == resValue {
					found = true
					break
				}
			}
			if !found {
				a.t.Fatalf("could not match header=%s", expectedHeader)
			}
		}
	}
}

func debugLog(prefix, header, msg string) {
	fmt.Printf("\n%s %s\n%s\n", prefix, header, msg)
}

func copyHttpResponse(response *http.Response) *http.Response {
	if response == nil {
		return nil
	}

	var resBodyBytes []byte
	if response.Body != nil {
		resBodyBytes, _ = ioutil.ReadAll(response.Body)
		response.Body = ioutil.NopCloser(bytes.NewBuffer(resBodyBytes))
	}

	resCopy := &http.Response{
		Header:        map[string][]string{},
		StatusCode:    response.StatusCode,
		Status:        response.Status,
		Body:          ioutil.NopCloser(bytes.NewBuffer(resBodyBytes)),
		Proto:         response.Proto,
		ProtoMinor:    response.ProtoMinor,
		ProtoMajor:    response.ProtoMajor,
		ContentLength: response.ContentLength,
	}

	for name, values := range response.Header {
		resCopy.Header[name] = values
	}

	return resCopy
}

func copyHttpRequest(request *http.Request) *http.Request {
	resCopy := &http.Request{
		Method:        request.Method,
		Host:          request.Host,
		Proto:         request.Proto,
		ProtoMinor:    request.ProtoMinor,
		ProtoMajor:    request.ProtoMajor,
		ContentLength: request.ContentLength,
	}

	if request.Body != nil {
		bodyBytes, _ := ioutil.ReadAll(request.Body)
		resCopy.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if request.URL != nil {
		r2URL := new(url.URL)
		*r2URL = *request.URL
		resCopy.URL = r2URL
	}

	headers := make(http.Header)
	for k, values := range request.Header {
		for _, hValue := range values {
			headers.Add(k, hValue)
		}
	}
	resCopy.Header = headers

	return resCopy
}

func quoted(in string) string {
	return string(fmt.Sprintf("%q", in))
}
