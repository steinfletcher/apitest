package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/steinfletcher/api-test/assert"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/textproto"
	"strings"
	"testing"
)

var divider = strings.Repeat("-", 10)
var requestDebugPrefix = fmt.Sprintf("%s>", divider)
var responseDebugPrefix = fmt.Sprintf("<%s", divider)

// APITest is the top level struct holding the test spec
type APITest struct {
	debugEnabled bool
	name         string
	request      *Request
	response     *Response
	observers    []Observe
	mocks        []*Mock
	t            *testing.T
	httpClient   *http.Client
	transport    *Transport
}

// Observe will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request)

// New creates a new api test. The name is optional and will appear in test reports
func New(name ...string) *APITest {
	apiTest := &APITest{}

	request := &Request{
		apiTest: apiTest,
		headers: map[string][]string{},
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

// Request returns the request spec
func (a *APITest) Request() *Request {
	return a.request
}

// Response returns the expected response
func (a *APITest) Response() *Response {
	return a.response
}

// Handler defines the http handler that is invoked when the test is run
func (a *APITest) Handler(handler http.Handler) *Request {
	a.request.handler = handler
	return a.request
}

// Request is the user defined request that will be invoked on the handler under test
type Request struct {
	handler         http.Handler
	interceptor     Intercept
	method          string
	url             string
	body            string
	query           map[string]string
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
func (r *Request) Intercept(interceptor Intercept) *Request {
	r.interceptor = interceptor
	return r
}

// Method is a builder method for setting the http method of the request
func (r *Request) Method(method string) *Request {
	r.method = method
	return r
}

// URL is a builder method for setting the url of the request
func (r *Request) URL(url string) *Request {
	r.url = url
	return r
}

// Get is a convenience method for setting the request as http.MethodGet
func (r *Request) Get(url string) *Request {
	r.method = http.MethodGet
	r.url = url
	return r
}

// Post is a convenience method for setting the request as http.MethodPost
func (r *Request) Post(url string) *Request {
	r.method = http.MethodPost
	r.url = url
	return r
}

// Put is a convenience method for setting the request as http.MethodPut
func (r *Request) Put(url string) *Request {
	r.method = http.MethodPut
	r.url = url
	return r
}

// Delete is a convenience method for setting the request as http.MethodDelete
func (r *Request) Delete(url string) *Request {
	r.method = http.MethodDelete
	r.url = url
	return r
}

// Patch is a convenience method for setting the request as http.MethodPatch
func (r *Request) Patch(url string) *Request {
	r.method = http.MethodPatch
	r.url = url
	return r
}

// Body is a builder method to set the request body
func (r *Request) Body(b string) *Request {
	r.body = b
	return r
}

// Query is a builder method to set the request query parameters.
// This can be used in combination with request.QueryCollection
func (r *Request) Query(q map[string]string) *Request {
	r.query = q
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

// Cookies is a builder method to set the request cookies
func (r *Request) Cookies(c ...*Cookie) *Request {
	r.cookies = c
	return r
}

// BasicAuth is a builder method to sets basic auth on the request.
// The credentials should be provided delimited by a colon, e.g. "username:password"
func (r *Request) BasicAuth(auth string) *Request {
	r.basicAuth = auth
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
	assert             Assert
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
	r.cookies = cookies
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
	r.assert = fn
	return r.apiTest.response
}

// End runs the test and all defined assertions
func (r *Response) End() {
	apiTest := r.apiTest

	if len(apiTest.mocks) > 0 {
		apiTest.transport = NewTransport(
			apiTest.mocks,
			apiTest.httpClient,
			r.apiTest.debugEnabled,
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
				observe(res.Result(), req)
			}
		}
	}()

	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)

	if a.response.assert != nil {
		err := a.response.assert(res.Result(), req)
		if err != nil {
			a.t.Fatal(err.Error())
		}
	}
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
			debug(requestDebugPrefix, "inbound http request", string(requestDump))
		}
	}

	a.request.handler.ServeHTTP(res, req)

	if a.debugEnabled {
		responseDump, err := httputil.DumpResponse(res.Result(), true)
		if err == nil {
			debug(responseDebugPrefix, "final response", string(responseDump))
		}
	}

	return res, req
}

func (a *APITest) BuildRequest() *http.Request {
	req, _ := http.NewRequest(a.request.method, a.request.url, bytes.NewBufferString(a.request.body))

	query := req.URL.Query()
	if a.request.queryCollection != nil {
		for _, param := range buildQueryCollection(a.request.queryCollection) {
			query.Add(param.l, param.r)
		}
	}

	if a.request.query != nil {
		for k, v := range a.request.query {
			query.Add(k, v)
		}
	}

	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}

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
		if isJSON(a.response.body) {
			assert.JsonEqual(a.t, a.response.body, res.Body.String())
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
			assert.Equal(a.t, 0, len(mismatchedFields), mismatchedFields...)
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

func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func debug(prefix, header, msg string) {
	fmt.Printf("\n%s %s\n%s\n", prefix, header, msg)
}
