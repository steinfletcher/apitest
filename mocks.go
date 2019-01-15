package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

var (
	ErrFailedToMatch = "failed to match any of the defined mocks"
)

type Transport struct {
	mocks           []*Mock
	nativeTransport http.RoundTripper
	httpClient      *http.Client
}

func NewTransport(mocks []*Mock, httpClient *http.Client) *Transport {
	t := &Transport{
		mocks:      mocks,
		httpClient: httpClient,
	}
	if httpClient != nil {
		t.nativeTransport = httpClient.Transport
	} else {
		t.nativeTransport = http.DefaultTransport
	}
	return t
}

func (r *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if matchedResponse := matches(req, r.mocks); matchedResponse != nil {
		return buildResponseFromMock(matchedResponse), nil
	}
	return nil, errors.New(ErrFailedToMatch)
}

func (r *Transport) Hijack() {
	if r.httpClient != nil {
		r.httpClient.Transport = r
		return
	}
	http.DefaultTransport = r
}

func (r *Transport) Reset() {
	if r.httpClient != nil {
		r.httpClient.Transport = r.nativeTransport
		return
	}
	http.DefaultTransport = r.nativeTransport
}

func buildResponseFromMock(mockResponse *MockResponse) *http.Response {
	res := &http.Response{
		Body:          ioutil.NopCloser(strings.NewReader(mockResponse.body)),
		Header:        mockResponse.headers,
		StatusCode:    mockResponse.statusCode,
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(mockResponse.body)),
	}

	for _, cookie := range mockResponse.cookies {
		if v := cookie.ToHttpCookie().String(); v != "" {
			res.Header.Add("Set-Cookie", v)
		}
	}

	return res
}

type Mock struct {
	isUsed   bool
	request  *MockRequest
	response *MockResponse
}

type MockRequest struct {
	mock         *Mock
	url          *url.URL
	method       string
	headers      map[string][]string
	query        map[string][]string
	queryPresent []string
	body         string
}

type MockResponse struct {
	mock       *Mock
	headers    map[string][]string
	cookies    []*Cookie
	body       string
	statusCode int
	times      int
}

func NewMock() *Mock {
	mock := &Mock{}
	req := &MockRequest{
		mock:    mock,
		headers: map[string][]string{},
		query:   map[string][]string{},
	}
	res := &MockResponse{
		mock:    mock,
		headers: map[string][]string{},
		times:   1,
	}
	mock.request = req
	mock.response = res
	return mock
}

func (m *Mock) Get(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodGet
	return m.request
}

func (m *Mock) Put(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPut
	return m.request
}

func (m *Mock) Post(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPost
	return m.request
}

func (m *Mock) Delete(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodDelete
	return m.request
}

func (m *Mock) Patch(u string) *MockRequest {
	m.parseUrl(u)
	m.request.method = http.MethodPatch
	return m.request
}

func (m *Mock) parseUrl(u string) {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	m.request.url = parsed
}

func (m *Mock) Method(method string) *MockRequest {
	m.request.method = method
	return m.request
}

func matches(req *http.Request, mocks []*Mock) *MockResponse {
	for _, mock := range mocks {
		if mock.isUsed {
			continue
		}

		matches := true
		for _, matcher := range matchers {
			if !matcher(req, mock.request) {
				matches = false
				break
			}
		}

		if matches {
			mock.isUsed = true
			return mock.response
		}
	}
	return nil
}

func (r *MockRequest) Body(b string) *MockRequest {
	r.body = b
	return r
}

func (r *MockRequest) Header(key, value string) *MockRequest {
	r.headers[key] = append(r.headers[key], value)
	return r
}

func (r *MockRequest) Headers(headers map[string]string) *MockRequest {
	for k, v := range headers {
		r.headers[k] = append(r.headers[k], v)
	}
	return r
}

func (r *MockRequest) Query(key, value string) *MockRequest {
	r.query[key] = append(r.query[key], value)
	return r
}

func (r *MockRequest) QueryParams(queryParams map[string]string) *MockRequest {
	for k, v := range queryParams {
		r.query[k] = append(r.query[k], v)
	}
	return r
}

func (r *MockRequest) QueryPresent(key string) *MockRequest {
	r.queryPresent = append(r.queryPresent, key)
	return r
}

func (r *MockRequest) RespondWith() *MockResponse {
	return r.mock.response
}

func (r *MockResponse) Header(name string, value string) *MockResponse {
	r.headers[name] = append(r.headers[name], value)
	return r
}

func (r *MockResponse) Headers(headers map[string]string) *MockResponse {
	for k, v := range headers {
		r.headers[k] = append(r.headers[k], v)
	}
	return r
}

func (r *MockResponse) Cookies(cookie ...*Cookie) *MockResponse {
	r.cookies = append(r.cookies, cookie...)
	return r
}

func (r *MockResponse) Cookie(name, value string) *MockResponse {
	r.cookies = append(r.cookies, NewCookie(name).Value(value))
	return r
}

func (r *MockResponse) Body(body string) *MockResponse {
	r.body = body
	return r
}

func (r *MockResponse) Status(statusCode int) *MockResponse {
	r.statusCode = statusCode
	return r
}

func (r *MockResponse) Times(times int) *MockResponse {
	r.times = times
	return r
}

func (r *MockResponse) End() *Mock {
	return r.mock
}

type Matcher func(*http.Request, *MockRequest) bool

var pathMatcher Matcher = func(r *http.Request, spec *MockRequest) bool {
	if r.URL.Path == spec.url.Path {
		return true
	}
	matched, err := regexp.MatchString(spec.url.Path, r.URL.Path)
	return matched && err == nil
}

var hostMatcher Matcher = func(r *http.Request, spec *MockRequest) bool {
	if spec.url.Host == "" {
		return true
	}
	if r.Host == spec.url.Host {
		return true
	}
	matched, err := regexp.MatchString(spec.url.Host, r.URL.Path)
	return matched && err != nil
}

var methodMatcher Matcher = func(r *http.Request, spec *MockRequest) bool {
	if r.Method == spec.method {
		return true
	}
	if spec.method == "" {
		return true
	}
	return false
}

var schemeMatcher Matcher = func(r *http.Request, spec *MockRequest) bool {
	if r.URL.Scheme == "" {
		return true
	}
	if spec.url.Scheme == "" {
		return true
	}
	return r.URL.Scheme == spec.url.Scheme
}

var headerMatcher = func(req *http.Request, spec *MockRequest) bool {
	for key, values := range spec.headers {
		var match bool
		var err error
		for _, field := range req.Header[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return false
				}
			}

			if match {
				break
			}
		}

		if !match {
			return false
		}
	}
	return true
}

var queryParamMatcher = func(req *http.Request, spec *MockRequest) bool {
	for key, values := range spec.query {
		var err error
		var match bool

		for _, field := range req.URL.Query()[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return false
				}
			}

			if match {
				break
			}
		}

		if !match {
			return false
		}
	}
	return true
}

var queryPresentMatcher = func(req *http.Request, spec *MockRequest) bool {
	for _, query := range spec.queryPresent {
		if req.URL.Query().Get(query) == "" {
			return false
		}
	}
	return true
}

var bodyMatcher = func(req *http.Request, spec *MockRequest) bool {
	if len(spec.body) == 0 {
		return true
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	if len(body) == 0 {
		return false
	}

	// replace body so it can be read again
	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Perform exact string match
	bodyStr := string(body)
	if bodyStr == spec.body {
		return true
	}

	// Perform regexp match
	match, _ := regexp.MatchString(spec.body, bodyStr)
	if match == true {
		return true
	}

	// Perform JSON match
	var reqJSON map[string]interface{}
	reqJSONErr := json.Unmarshal(body, &reqJSON)

	var matchJSON map[string]interface{}
	specJSONErr := json.Unmarshal([]byte(spec.body), &matchJSON)

	isJSON := reqJSONErr == nil && specJSONErr == nil
	if isJSON && reflect.DeepEqual(reqJSON, matchJSON) {
		return true
	}

	return false
}

var matchers = []Matcher{
	pathMatcher,
	hostMatcher,
	schemeMatcher,
	methodMatcher,
	headerMatcher,
	queryParamMatcher,
	queryPresentMatcher,
	bodyMatcher,
}
