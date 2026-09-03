package apitest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"sync"
)

// Mock represents the entire interaction for a mock to be used for testing
type Mock struct {
	m               *sync.Mutex
	isUsed          bool
	request         *MockRequest
	response        *MockResponse
	httpClient      *http.Client
	debugStandalone bool
	times           int
	timesSet        bool
	anyTimesSet     bool
}

// Matches checks whether the given request matches the mock
func (m *Mock) Matches(req *http.Request) []error {
	var errs []error
	for _, matcher := range m.request.matchers {
		if matcherError := matcher(req, m.request); matcherError != nil {
			errs = append(errs, matcherError)
		}
	}
	return errs
}

func (m *Mock) copy() *Mock {
	newMock := *m

	newMock.m = &sync.Mutex{}

	req := *m.request
	req.mock = &newMock
	newMock.request = &req

	newMock.response = m.response.deepCopy()
	newMock.response.mock = &newMock

	return &newMock
}

// MockRequest represents the http request side of a mock interaction
type MockRequest struct {
	mock               *Mock
	url                *url.URL
	method             string
	headers            map[string][]string
	basicAuthUsername  string
	basicAuthPassword  string
	headerPresent      []string
	headerNotPresent   []string
	formData           map[string][]string
	formDataPresent    []string
	formDataNotPresent []string
	query              map[string][]string
	queryPresent       []string
	queryNotPresent    []string
	cookie             []Cookie
	cookiePresent      []string
	cookieNotPresent   []string
	body               string
	bodyRegexp         string
	matchers           []Matcher
}

// UnmatchedMock exposes some information about mocks that failed to match a request
type UnmatchedMock struct {
	URL url.URL
}

// MockResponse represents the http response side of a mock interaction
type MockResponse struct {
	mock             *Mock
	timeout          bool
	headers          map[string][]string
	cookies          []*Cookie
	body             string
	statusCode       int
	fixedDelayMillis int64
	mu               sync.RWMutex // Add a mutex for thread-safe access
}

func (r *MockResponse) deepCopy() *MockResponse {
	newResponse := &MockResponse{
		timeout:          r.timeout,
		headers:          make(map[string][]string),
		cookies:          make([]*Cookie, len(r.cookies)),
		body:             r.body,
		statusCode:       r.statusCode,
		fixedDelayMillis: r.fixedDelayMillis,
		mu:               sync.RWMutex{},
	}

	for k, v := range r.headers {
		newHeader := make([]string, len(v))
		copy(newHeader, v)
		newResponse.headers[k] = newHeader
	}

	for i, cookie := range r.cookies {
		newCookie := *cookie
		newResponse.cookies[i] = &newCookie
	}

	return newResponse
}

// StandaloneMocks for using mocks outside of API tests context
type StandaloneMocks struct {
	mocks      []*Mock
	httpClient *http.Client
	debug      bool
}

// NewStandaloneMocks create a series of StandaloneMocks
func NewStandaloneMocks(mocks ...*Mock) *StandaloneMocks {
	return &StandaloneMocks{
		mocks: mocks,
	}
}

// HttpClient use the given http client
func (r *StandaloneMocks) HttpClient(cli *http.Client) *StandaloneMocks {
	r.httpClient = cli
	return r
}

// Debug switch on debugging mode
func (r *StandaloneMocks) Debug() *StandaloneMocks {
	r.debug = true
	return r
}

// End finalises the mock, ready for use
func (r *StandaloneMocks) End() func() {
	transport := newTransport(
		r.mocks,
		r.httpClient,
		r.debug,
		false,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

// NewMock create a new mock, ready for configuration using the builder pattern
func NewMock() *Mock {
	mock := &Mock{
		m:     &sync.Mutex{},
		times: 1,
	}
	mock.request = &MockRequest{
		mock:     mock,
		headers:  map[string][]string{},
		formData: map[string][]string{},
		query:    map[string][]string{},
		matchers: defaultMatchers,
	}
	mock.response = &MockResponse{
		mock:    mock,
		headers: map[string][]string{},
	}
	return mock
}

// Debug is used to set debug mode for mocks in standalone mode.
// This is overridden by the debug setting in the `APITest` struct
func (m *Mock) Debug() *Mock {
	m.debugStandalone = true
	return m
}

// HttpClient allows the developer to provide a custom http client when using mocks
func (m *Mock) HttpClient(cli *http.Client) *Mock {
	m.httpClient = cli
	return m
}

// Get configures the mock to match http method GET
func (m *Mock) Get(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodGet
	return m.request
}

// Getf configures the mock to match http method GET and supports formatting
func (m *Mock) Getf(format string, args ...any) *MockRequest {
	return m.Get(fmt.Sprintf(format, args...))
}

// Head configures the mock to match http method HEAD
func (m *Mock) Head(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodHead
	return m.request
}

// Put configures the mock to match http method PUT
func (m *Mock) Put(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPut
	return m.request
}

// Putf configures the mock to match http method PUT and supports formatting
func (m *Mock) Putf(format string, args ...any) *MockRequest {
	return m.Put(fmt.Sprintf(format, args...))
}

// Post configures the mock to match http method POST
func (m *Mock) Post(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPost
	return m.request
}

// Postf configures the mock to match http method POST and supports formatting
func (m *Mock) Postf(format string, args ...any) *MockRequest {
	return m.Post(fmt.Sprintf(format, args...))
}

// Delete configures the mock to match http method DELETE
func (m *Mock) Delete(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodDelete
	return m.request
}

// Deletef configures the mock to match http method DELETE and supports formatting
func (m *Mock) Deletef(format string, args ...any) *MockRequest {
	return m.Delete(fmt.Sprintf(format, args...))
}

// Patch configures the mock to match http method PATCH
func (m *Mock) Patch(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPatch
	return m.request
}

// Patchf configures the mock to match http method PATCH and supports formatting
func (m *Mock) Patchf(format string, args ...any) *MockRequest {
	return m.Patch(fmt.Sprintf(format, args...))
}

func (m *Mock) parseUrl(u string) {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	m.request.url = parsed
}

// Method configures mock to match given http method
func (m *Mock) Method(method string) *MockRequest {
	m.request.method = method
	return m.request
}

// Body configures the mock request to match the given body
func (r *MockRequest) Body(b string) *MockRequest {
	r.body = b
	return r
}

// BodyRegexp configures the mock request to match the given body using the regexp matcher
func (r *MockRequest) BodyRegexp(b string) *MockRequest {
	r.bodyRegexp = b
	return r
}

// Bodyf configures the mock request to match the given body. Supports formatting the body
func (r *MockRequest) Bodyf(format string, args ...any) *MockRequest {
	return r.Body(fmt.Sprintf(format, args...))
}

// BodyFromFile configures the mock request to match the given body from a file
func (r *MockRequest) BodyFromFile(f string) *MockRequest {
	b, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}
	r.body = string(b)
	return r
}

// JSON is a convenience method for setting the mock request body
func (r *MockRequest) JSON(v any) *MockRequest {
	switch x := v.(type) {
	case string:
		r.body = x
	case []byte:
		r.body = string(x)
	default:
		asJSON, _ := json.Marshal(x)
		r.body = string(asJSON)
	}
	return r
}

// Header configures the mock request to match the given header
func (r *MockRequest) Header(key, value string) *MockRequest {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers configures the mock request to match the given headers
func (r *MockRequest) Headers(headers map[string]string) *MockRequest {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

// HeaderPresent configures the mock request to match when this header is present, regardless of value
func (r *MockRequest) HeaderPresent(key string) *MockRequest {
	r.headerPresent = append(r.headerPresent, key)
	return r
}

// HeaderNotPresent configures the mock request to match when the header is not present
func (r *MockRequest) HeaderNotPresent(key string) *MockRequest {
	r.headerNotPresent = append(r.headerNotPresent, key)
	return r
}

// BasicAuth configures the mock request to match the given basic auth parameters
func (r *MockRequest) BasicAuth(username, password string) *MockRequest {
	r.basicAuthUsername = username
	r.basicAuthPassword = password
	return r
}

// FormData configures the mock request to math the given form data
func (r *MockRequest) FormData(key string, values ...string) *MockRequest {
	r.formData[key] = append(r.formData[key], values...)
	return r
}

// FormDataPresent configures the mock request to match when the form data is present, regardless of values
func (r *MockRequest) FormDataPresent(key string) *MockRequest {
	r.formDataPresent = append(r.formDataPresent, key)
	return r
}

// FormDataNotPresent configures the mock request to match when the form data is not present
func (r *MockRequest) FormDataNotPresent(key string) *MockRequest {
	r.formDataNotPresent = append(r.formDataNotPresent, key)
	return r
}

// Query configures the mock request to match a query param
func (r *MockRequest) Query(key, value string) *MockRequest {
	r.query[key] = append(r.query[key], value)
	return r
}

// QueryParams configures the mock request to match a number of query params
func (r *MockRequest) QueryParams(queryParams map[string]string) *MockRequest {
	for k, v := range queryParams {
		r.query[k] = append(r.query[k], v)
	}
	return r
}

// QueryCollection configures the mock request to match a number of repeating query params, e.g. ?a=1&a=2&a=3
func (r *MockRequest) QueryCollection(queryParams map[string][]string) *MockRequest {
	for k, v := range queryParams {
		r.query[k] = append(r.query[k], v...)
	}
	return r
}

// QueryPresent configures the mock request to match when a query param is present, regardless of value
func (r *MockRequest) QueryPresent(key string) *MockRequest {
	r.queryPresent = append(r.queryPresent, key)
	return r
}

// QueryNotPresent configures the mock request to match when the query param is not present
func (r *MockRequest) QueryNotPresent(key string) *MockRequest {
	r.queryNotPresent = append(r.queryNotPresent, key)
	return r
}

// Cookie configures the mock request to match a cookie
func (r *MockRequest) Cookie(name, value string) *MockRequest {
	r.cookie = append(r.cookie, Cookie{name: &name, value: &value})
	return r
}

// CookiePresent configures the mock request to match when a cookie is present, regardless of value
func (r *MockRequest) CookiePresent(name string) *MockRequest {
	r.cookiePresent = append(r.cookiePresent, name)
	return r
}

// CookieNotPresent configures the mock request to match when a cookie is not present
func (r *MockRequest) CookieNotPresent(name string) *MockRequest {
	r.cookieNotPresent = append(r.cookieNotPresent, name)
	return r
}

// AddMatcher configures the mock request to match using a custom matcher
func (r *MockRequest) AddMatcher(matcher Matcher) *MockRequest {
	r.matchers = append(r.matchers, matcher)
	return r
}

// RespondWith finalises the mock request phase of set up and allowing the definition of response attributes to be defined
func (r *MockRequest) RespondWith() *MockResponse {
	return r.mock.response
}

// Timeout forces the mock to return a http timeout
func (r *MockResponse) Timeout() *MockResponse {
	r.timeout = true
	return r
}

// Header respond with the given header
func (r *MockResponse) Header(key string, value string) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers respond with the given headers
func (r *MockResponse) Headers(headers map[string]string) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

// Cookies respond with the given cookies
func (r *MockResponse) Cookies(cookie ...*Cookie) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cookies = append(r.cookies, cookie...)
	return r
}

// Cookie respond with the given cookie
func (r *MockResponse) Cookie(name, value string) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cookies = append(r.cookies, NewCookie(name).Value(value))
	return r
}

// Body sets the mock response body
func (r *MockResponse) Body(body string) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.body = body
	return r
}

// Bodyf sets the mock response body. Supports formatting
func (r *MockResponse) Bodyf(format string, args ...any) *MockResponse {
	return r.Body(fmt.Sprintf(format, args...))
}

// BodyFromFile defines the mock response body from a file
func (r *MockResponse) BodyFromFile(f string) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}
	r.body = string(b)
	return r
}

// JSON is a convenience method for setting the mock response body
func (r *MockResponse) JSON(v any) *MockResponse {
	switch x := v.(type) {
	case string:
		r.body = x
	case []byte:
		r.body = string(x)
	default:
		asJSON, _ := json.Marshal(x)
		r.body = string(asJSON)
	}
	return r
}

// Status respond with the given status
func (r *MockResponse) Status(statusCode int) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.statusCode = statusCode
	return r
}

// FixedDelay will return the response after the given number of milliseconds.
// APITest::EnableMockResponseDelay must be set for this to take effect.
// If Timeout is set this has no effect.
func (r *MockResponse) FixedDelay(delay int64) *MockResponse {
	r.fixedDelayMillis = delay
	return r
}

// Times respond the given number of times, if AnyTimes is set this has no effect
func (r *MockResponse) Times(times int) *MockResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mock.times = times
	r.mock.timesSet = true
	return r
}

// AnyTimes respond any number of times
func (r *MockResponse) AnyTimes() *MockResponse {
	r.mock.anyTimesSet = true
	return r
}

// End finalise the response definition phase in order for the mock to be used
func (r *MockResponse) End() *Mock {
	return r.mock
}

// EndStandalone finalises the response definition of standalone mocks
func (r *MockResponse) EndStandalone(other ...*Mock) func() {
	transport := newTransport(
		append([]*Mock{r.mock}, other...),
		r.mock.httpClient,
		r.mock.debugStandalone,
		false,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

func (timeoutError) Error() string   { return "deadline exceeded" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
