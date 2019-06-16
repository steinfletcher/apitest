package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type Transport struct {
	debugEnabled    bool
	mocks           []*Mock
	nativeTransport http.RoundTripper
	httpClient      *http.Client
	observe         Observe
	apiTest         *APITest
}

func newTransport(
	mocks []*Mock,
	httpClient *http.Client,
	debugEnabled bool,
	observe Observe,
	apiTest *APITest) *Transport {

	t := &Transport{
		mocks:        mocks,
		httpClient:   httpClient,
		debugEnabled: debugEnabled,
		observe:      observe,
		apiTest:      apiTest,
	}
	if httpClient != nil {
		t.nativeTransport = httpClient.Transport
	} else {
		t.nativeTransport = http.DefaultTransport
	}
	return t
}

type unmatchedMockError struct {
	errors map[int][]error
}

func newUnmatchedMockError() *unmatchedMockError {
	return &unmatchedMockError{
		errors: map[int][]error{},
	}
}

func (u *unmatchedMockError) addErrors(mockNumber int, errors ...error) *unmatchedMockError {
	u.errors[mockNumber] = append(u.errors[mockNumber], errors...)
	return u
}

func (u *unmatchedMockError) Error() string {
	var strBuilder strings.Builder
	strBuilder.WriteString("received request did not match any mocks\n\n")
	for _, mockNumber := range u.orderedMockKeys() {
		strBuilder.WriteString(fmt.Sprintf("Mock %d mismatches:\n", mockNumber))
		for _, err := range u.errors[mockNumber] {
			strBuilder.WriteString("• ")
			strBuilder.WriteString(err.Error())
			strBuilder.WriteString("\n")
		}
		strBuilder.WriteString("\n")
	}
	return strBuilder.String()
}

func (u *unmatchedMockError) orderedMockKeys() []int {
	var mockKeys []int
	for mockKey := range u.errors {
		mockKeys = append(mockKeys, mockKey)
	}
	sort.Ints(mockKeys)
	return mockKeys
}

// RoundTrip implementation intended to match a given expected mock request or throw an error with a list of reasons why no match was found.
func (r *Transport) RoundTrip(req *http.Request) (mockResponse *http.Response, matchErrors error) {
	if r.debugEnabled {
		defer func() {
			debugMock(mockResponse, req)
		}()
	}

	if r.observe != nil {
		defer func() {
			r.observe(mockResponse, req, r.apiTest)
		}()
	}

	matchedResponse, matchErrors := matches(req, r.mocks)
	if matchErrors == nil {
		return buildResponseFromMock(matchedResponse), nil
	}

	if r.debugEnabled {
		fmt.Printf("failed to match mocks. Errors: %s\n", matchErrors)
	}

	return nil, matchErrors
}

func debugMock(res *http.Response, req *http.Request) {
	requestDump, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		debugLog(requestDebugPrefix, "request to mock", string(requestDump))
	}

	if res != nil {
		responseDump, err := httputil.DumpResponse(res, true)
		if err == nil {
			debugLog(responseDebugPrefix, "response from mock", string(responseDump))
		}
	} else {
		debugLog(responseDebugPrefix, "response from mock", "")
	}
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
	if mockResponse == nil {
		return nil
	}

	contentTypeHeader := mockResponse.headers["Content-Type"]
	var contentType string

	// if the content type isn't set and the body contains json, set content type as json
	if len(mockResponse.body) > 0 {
		if len(contentTypeHeader) == 0 {
			if json.Valid([]byte(mockResponse.body)) {
				contentType = "application/json"
			} else {
				contentType = "text/plain"
			}
		} else {
			contentType = contentTypeHeader[0]
		}
	}

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

	if contentType != "" {
		res.Header.Set("Content-Type", contentType)
	}

	return res
}

type Mock struct {
	isUsed          bool
	request         *MockRequest
	response        *MockResponse
	httpClient      *http.Client
	debugStandalone bool
}

type MockRequest struct {
	mock             *Mock
	url              *url.URL
	method           string
	headers          map[string][]string
	headerPresent    []string
	headerNotPresent []string
	query            map[string][]string
	queryPresent     []string
	queryNotPresent  []string
	cookie           []Cookie
	cookiePresent    []string
	cookieNotPresent []string
	body             string
	matchers         []Matcher
}

type MockResponse struct {
	mock       *Mock
	headers    map[string][]string
	cookies    []*Cookie
	body       string
	statusCode int
	times      int
}

type StandaloneMocks struct {
	mocks      []*Mock
	httpClient *http.Client
	debug      bool
}

func NewStandaloneMocks(mocks ...*Mock) *StandaloneMocks {
	return &StandaloneMocks{
		mocks: mocks,
	}
}

func (r *StandaloneMocks) HttpClient(cli *http.Client) *StandaloneMocks {
	r.httpClient = cli
	return r
}

func (r *StandaloneMocks) Debug() *StandaloneMocks {
	r.debug = true
	return r
}

func (r *StandaloneMocks) End() func() {
	transport := newTransport(
		r.mocks,
		r.httpClient,
		r.debug,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

func NewMock() *Mock {
	mock := &Mock{}
	req := &MockRequest{
		mock:     mock,
		headers:  map[string][]string{},
		query:    map[string][]string{},
		matchers: defaultMatchers,
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

func matches(req *http.Request, mocks []*Mock) (*MockResponse, error) {
	mockError := newUnmatchedMockError()
	for mockNumber, mock := range mocks {
		if mock.isUsed {
			continue
		}

		var mockMatchErrors []error
		for _, matcher := range mock.request.matchers {
			if matcherError := matcher(req, mock.request); matcherError != nil {
				mockMatchErrors = append(mockMatchErrors, matcherError)
			}
		}

		if len(mockMatchErrors) == 0 {
			mock.isUsed = true
			return mock.response, nil
		}

		mockError = mockError.addErrors(mockNumber+1, mockMatchErrors...)
	}

	return nil, mockError
}

func (r *MockRequest) Body(b string) *MockRequest {
	r.body = b
	return r
}

func (r *MockRequest) Header(key, value string) *MockRequest {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

func (r *MockRequest) Headers(headers map[string]string) *MockRequest {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

func (r *MockRequest) HeaderPresent(key string) *MockRequest {
	r.headerPresent = append(r.headerPresent, key)
	return r
}

func (r *MockRequest) HeaderNotPresent(key string) *MockRequest {
	r.headerNotPresent = append(r.headerNotPresent, key)
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

func (r *MockRequest) QueryNotPresent(key string) *MockRequest {
	r.queryNotPresent = append(r.queryNotPresent, key)
	return r
}

func (r *MockRequest) Cookie(name, value string) *MockRequest {
	r.cookie = append(r.cookie, Cookie{name: &name, value: &value})
	return r
}

func (r *MockRequest) CookiePresent(name string) *MockRequest {
	r.cookiePresent = append(r.cookiePresent, name)
	return r
}

func (r *MockRequest) CookieNotPresent(name string) *MockRequest {
	r.cookieNotPresent = append(r.cookieNotPresent, name)
	return r
}

func (r *MockRequest) AddMatcher(matcher Matcher) *MockRequest {
	r.matchers = append(r.matchers, matcher)
	return r
}

func (r *MockRequest) RespondWith() *MockResponse {
	return r.mock.response
}

func (r *MockResponse) Header(key string, value string) *MockResponse {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

func (r *MockResponse) Headers(headers map[string]string) *MockResponse {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
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

func (r *MockResponse) EndStandalone(other ...*Mock) func() {
	transport := newTransport(
		append([]*Mock{r.mock}, other...),
		r.mock.httpClient,
		r.mock.debugStandalone,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

// Matcher type accepts the actual request and a mock request to match against.
// Will return an error that describes why there was a mismatch if the inputs do not match or nil if they do.
type Matcher func(*http.Request, *MockRequest) error

var pathMatcher Matcher = func(r *http.Request, spec *MockRequest) error {
	receivedPath := r.URL.Path
	mockPath := spec.url.Path
	if receivedPath == mockPath {
		return nil
	}
	matched, err := regexp.MatchString(mockPath, receivedPath)
	return errorOrNil(matched && err == nil, func() string {
		return fmt.Sprintf("received path %s did not match mock path %s", receivedPath, mockPath)
	})
}

var hostMatcher Matcher = func(r *http.Request, spec *MockRequest) error {
	receivedHost := r.Host
	if receivedHost == "" {
		receivedHost = r.URL.Host
	}
	mockHost := spec.url.Host
	if mockHost == "" {
		return nil
	}
	if receivedHost == mockHost {
		return nil
	}
	matched, err := regexp.MatchString(mockHost, r.URL.Path)
	return errorOrNil(matched && err != nil, func() string {
		return fmt.Sprintf("received host %s did not match mock host %s", receivedHost, mockHost)
	})
}

var methodMatcher Matcher = func(r *http.Request, spec *MockRequest) error {
	receivedMethod := r.Method
	mockMethod := spec.method
	if receivedMethod == mockMethod {
		return nil
	}
	if mockMethod == "" {
		return nil
	}
	return fmt.Errorf("received method %s did not match mock method %s", receivedMethod, mockMethod)
}

var schemeMatcher Matcher = func(r *http.Request, spec *MockRequest) error {
	receivedScheme := r.URL.Scheme
	mockScheme := spec.url.Scheme
	if receivedScheme == "" {
		return nil
	}
	if mockScheme == "" {
		return nil
	}
	return errorOrNil(receivedScheme == mockScheme, func() string {
		return fmt.Sprintf("received scheme %s did not match mock scheme %s", receivedScheme, mockScheme)
	})
}

var headerMatcher = func(req *http.Request, spec *MockRequest) error {
	mockHeaders := spec.headers
	for key, values := range mockHeaders {
		var match bool
		var err error
		receivedHeaders := req.Header
		for _, field := range receivedHeaders[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return fmt.Errorf("unable to match received header value %s against expected value %s", value, field)
				}
			}

			if match {
				break
			}
		}

		if !match {
			return fmt.Errorf("not all of received headers %s matched expected mock headers %s", receivedHeaders, mockHeaders)
		}
	}
	return nil
}

var headerPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, header := range spec.headerPresent {
		if req.Header.Get(header) == "" {
			return fmt.Errorf("expected header '%s' was not present", header)
		}
	}
	return nil
}

var headerNotPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, header := range spec.headerNotPresent {
		if req.Header.Get(header) != "" {
			return fmt.Errorf("unexpected header '%s' was present", header)
		}
	}
	return nil
}

var queryParamMatcher = func(req *http.Request, spec *MockRequest) error {
	mockQueryParams := spec.query
	for key, values := range mockQueryParams {
		var err error
		var match bool

		receivedQueryParams := req.URL.Query()

		for _, field := range receivedQueryParams[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return fmt.Errorf("unable to match received query param value %s against expected value %s", value, field)
				}
			}

			if match {
				break
			}
		}

		if !match {
			return fmt.Errorf("not all of received query params %s matched expected mock query params %s", receivedQueryParams, mockQueryParams)
		}
	}
	return nil
}

var queryPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, query := range spec.queryPresent {
		if req.URL.Query().Get(query) == "" {
			return fmt.Errorf("expected query param %s not received", query)
		}
	}
	return nil
}

var queryNotPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, query := range spec.queryNotPresent {
		if req.URL.Query().Get(query) != "" {
			return fmt.Errorf("unexpected query param '%s' present", query)
		}
	}
	return nil
}

var cookieMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, c := range spec.cookie {
		foundCookie, _ := req.Cookie(*c.name)
		if foundCookie == nil {
			return fmt.Errorf("expected cookie with name '%s' not received", *c.name)
		}
		if _, mismatches := compareCookies(&c, foundCookie); len(mismatches) > 0 {
			return fmt.Errorf("failed to match cookie: %v", mismatches)
		}
	}
	return nil
}

var cookiePresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, c := range spec.cookiePresent {
		foundCookie, _ := req.Cookie(c)
		if foundCookie == nil {
			return fmt.Errorf("expected cookie with name '%s' not received", c)
		}
	}
	return nil
}

var cookieNotPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	for _, c := range spec.cookieNotPresent {
		foundCookie, _ := req.Cookie(c)
		if foundCookie != nil {
			return fmt.Errorf("did not expect a cookie with name '%s'", c)
		}
	}
	return nil
}

var bodyMatcher = func(req *http.Request, spec *MockRequest) error {
	mockBody := spec.body

	if len(mockBody) == 0 {
		return nil
	}

	if req.Body == nil {
		return errors.New("expected a body but received none")
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("expected a body but received none")
	}

	// replace body so it can be read again
	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Perform exact string match
	bodyStr := string(body)
	if bodyStr == mockBody {
		return nil
	}

	// Perform regexp match
	match, _ := regexp.MatchString(mockBody, bodyStr)
	if match == true {
		return nil
	}

	// Perform JSON match
	var reqJSON map[string]interface{}
	reqJSONErr := json.Unmarshal(body, &reqJSON)

	var matchJSON map[string]interface{}
	specJSONErr := json.Unmarshal([]byte(mockBody), &matchJSON)

	isJSON := reqJSONErr == nil && specJSONErr == nil
	if isJSON && reflect.DeepEqual(reqJSON, matchJSON) {
		return nil
	}

	return fmt.Errorf("received body %s did not match expected mock body %s", bodyStr, mockBody)
}

func errorOrNil(statement bool, errorMessage func() string) error {
	if statement {
		return nil
	}
	return errors.New(errorMessage())
}

var defaultMatchers = []Matcher{
	pathMatcher,
	hostMatcher,
	schemeMatcher,
	methodMatcher,
	headerMatcher,
	headerPresentMatcher,
	headerNotPresentMatcher,
	queryParamMatcher,
	queryPresentMatcher,
	queryNotPresentMatcher,
	bodyMatcher,
	cookieMatcher,
	cookiePresentMatcher,
	cookieNotPresentMatcher,
}
