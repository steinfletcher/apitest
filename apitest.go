package apitest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
)

// APITest is the top level struct holding the test spec
type APITest struct {
	handler  http.Handler
	request  *Request
	response *Response
	observer Observe
	t        *testing.T
}

// New creates a new api test with the given http.Handler
func New(handler http.Handler) *Request {
	apiTest := &APITest{}

	request := &Request{apiTest: apiTest}
	response := &Response{apiTest: apiTest}
	apiTest.request = request
	apiTest.response = response
	apiTest.handler = handler

	return apiTest.request
}

// Observer will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request)

// Request is the user defined request that will be invoked on the handler under test
type Request struct {
	method          string
	url             string
	body            string
	query           map[string]string
	queryCollection map[string][]string
	headers         map[string]string
	cookies         map[string]string
	basicAuth       string
	apiTest         *APITest
}

type pair struct {
	l string
	r string
}

var DumpHttp Observe = func(res *http.Response, req *http.Request) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err == nil {
		fmt.Println("--> http request dump\n\n" + string(requestDump))
	}

	responseDump, err := httputil.DumpResponse(res, true)
	if err == nil {
		fmt.Println("<-- http response dump:\n\n" + string(responseDump))
	}
}

// Observe is a builder method for setting the observer
func (r *Request) Observe(observer Observe) *Request {
	r.apiTest.observer = observer
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

// Query is a builder method to set the request query parameters
// This can be used in combination with request.Query
func (r *Request) QueryCollection(q map[string][]string) *Request {
	r.queryCollection = q
	return r
}

// Headers is a builder method to set the request headers
func (r *Request) Headers(h map[string]string) *Request {
	r.headers = h
	return r
}

// Headers is a builder method to set the request headers
func (r *Request) Cookies(c map[string]string) *Request {
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
	headers            map[string]string
	cookies            map[string]string
	cookiesPresent     []string
	httpCookies        []http.Cookie
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
func (r *Response) Cookies(cookies map[string]string) *Response {
	r.cookies = cookies
	return r
}

// HttpCookies is the expected response cookies
func (r *Response) HttpCookies(cookies []http.Cookie) *Response {
	r.httpCookies = cookies
	return r
}

// CookiePresent is used to assert that a cookie is present in the response,
// regardless of its value
func (r *Response) CookiePresent(cookieName string) *Response {
	r.cookiesPresent = append(r.cookiesPresent, cookieName)
	return r
}

// Headers is the expected response headers
func (r *Response) Headers(headers map[string]string) *Response {
	r.headers = headers
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

// JSONPath provides support for jsonpath expectations as defined by https://goessner.net/articles/JsonPath/
func (r *Response) JSONPath(expression string, assert func(interface{})) *Response {
	r.jsonPathExpression = expression
	r.jsonPathAssert = assert
	return r.apiTest.response
}

// End runs the test and all defined assertions
func (r *Response) End() {
	r.apiTest.run()
}

func (a *APITest) run() {
	res, req := a.runTest()
	if a.observer != nil {
		a.observer(res.Result(), req)
	}
	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)
	a.assertJSONPath(res)
	if a.response.assert != nil {
		err := a.response.assert(res.Result(), req)
		if err != nil {
			a.t.Fatal(err.Error())
		}
	}
}

func (a *APITest) runTest() (*httptest.ResponseRecorder, *http.Request) {
	req := a.buildRequestFromTestCase()
	res := httptest.NewRecorder()
	a.handler.ServeHTTP(res, req)
	return res, req
}

func (a *APITest) buildRequestFromTestCase() *http.Request {
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
		req.Header.Set(k, v)
	}

	for k, v := range a.request.cookies {
		cookie := &http.Cookie{Name: k, Value: v}
		req.AddCookie(cookie)
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
		assert.Equal(a.t, a.response.status, res.Code)
	}

	if a.response.body != "" {
		if isJSON(a.response.body) {
			assert.JSONEq(a.t, a.response.body, res.Body.String())
		} else {
			assert.Equal(a.t, a.response.body, res.Body.String())
		}
	}
}

func (a *APITest) assertCookies(response *httptest.ResponseRecorder) {
	if a.response.cookies != nil {
		for name, value := range a.response.cookies {
			foundCookie := false
			for _, cookie := range getResponseCookies(response) {
				if cookie.Name == name && cookie.Value == value {
					foundCookie = true
				}
			}
			assert.Equal(a.t, true, foundCookie, "Cookie not found - "+name)
		}
	}

	if len(a.response.cookiesPresent) > 0 {
		for _, cookieName := range a.response.cookiesPresent {
			foundCookie := false
			for _, cookie := range getResponseCookies(response) {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			assert.Equal(a.t, true, foundCookie, "Cookie not found - "+cookieName)
		}
	}

	if len(a.response.httpCookies) > 0 {
		for _, httpCookie := range a.response.httpCookies {
			foundCookie := false
			for _, cookie := range getResponseCookies(response) {
				if compareHttpCookies(cookie, &httpCookie) {
					foundCookie = true
				}
			}
			assert.Equal(a.t, true, foundCookie, "Cookie not found - "+httpCookie.Name)
		}
	}
}

// only compare a subset of fields for flexibility
func compareHttpCookies(l *http.Cookie, r *http.Cookie) bool {
	return l.Name == r.Name &&
		l.Value == r.Value &&
		l.Domain == r.Domain &&
		l.Expires == r.Expires &&
		l.MaxAge == r.MaxAge &&
		l.Secure == r.Secure &&
		l.HttpOnly == r.HttpOnly &&
		l.SameSite == r.SameSite
}

func getResponseCookies(response *httptest.ResponseRecorder) []*http.Cookie {
	for _, rawCookieString := range response.Result().Header["Set-Cookie"] {
		rawRequest := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", rawCookieString)
		req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(rawRequest)))
		if err != nil {
			panic("failed to parse response cookies. error: " + err.Error())
		}
		return req.Cookies()
	}
	return []*http.Cookie{}
}

func (a *APITest) assertHeaders(res *httptest.ResponseRecorder) {
	if a.response.headers != nil {
		for k, v := range a.response.headers {
			header := res.Header().Get(k)
			assert.Equal(a.t, v, header, fmt.Sprintf("'%s' header should be equal", k))
		}
	}
}

func (a *APITest) assertJSONPath(res *httptest.ResponseRecorder) {
	if a.response.jsonPathExpression != "" {
		v := interface{}(nil)
		err := json.Unmarshal(res.Body.Bytes(), &v)

		value, err := jsonpath.Get(a.response.jsonPathExpression, v)
		if err != nil {
			assert.Nil(a.t, err)
		}

		a.response.jsonPathAssert(value.(interface{}))
	}
}

func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
