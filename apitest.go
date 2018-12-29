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
	"strings"
	"testing"
)

type APITest struct {
	name     string
	handler  http.Handler
	request  *Request
	response *Response
	t        *testing.T
}

func New(handler http.Handler) *Request {
	apiTest := &APITest{}

	request := &Request{apiTest: apiTest}
	response := &Response{apiTest: apiTest}
	apiTest.request = request
	apiTest.response = response
	apiTest.handler = handler

	return apiTest.request
}

type Handler struct {
	handler http.Handler
	apiTest *APITest
}

func (r *Request) Name(n string) *Request {
	r.apiTest.name = n
	return r.apiTest.request
}

type Request struct {
	method    string
	url       string
	bodyJSON  string
	body      string
	headers   map[string]string
	query     map[string]string
	cookies   map[string]string
	basicAuth string
	apiTest   *APITest
}

func (r *Request) Get(url string) *Request {
	r.method = http.MethodGet
	r.url = url
	return r
}

func (r *Request) Post(url string) *Request {
	r.method = http.MethodPost
	r.url = url
	return r
}

func (r *Request) Put(url string) *Request {
	r.method = http.MethodPut
	r.url = url
	return r
}

func (r *Request) Delete(url string) *Request {
	r.method = http.MethodDelete
	r.url = url
	return r
}

func (r *Request) Patch(url string) *Request {
	r.method = http.MethodPatch
	r.url = url
	return r
}

func (r *Request) BodyJSON(b string) *Request {
	r.bodyJSON = b
	return r
}

func (r *Request) Body(b string) *Request {
	r.body = b
	return r
}

func (r *Request) Query(q map[string]string) *Request {
	r.query = q
	return r
}

func (r *Request) Headers(h map[string]string) *Request {
	r.headers = h
	return r
}

func (r *Request) Cookies(c map[string]string) *Request {
	r.cookies = c
	return r
}

func (r *Request) BasicAuth(auth string) *Request {
	r.basicAuth = auth
	return r
}

func (r *Request) Expect(t *testing.T) *Response {
	r.apiTest.t = t
	return r.apiTest.response
}

type Response struct {
	status             int
	bodyJSON           string
	body               string
	headers            map[string]string
	cookies            map[string]string
	cookiesPresent     []string
	jsonPathExpression string
	jsonPathAssert     func(interface{})
	apiTest            *APITest
	assert             func(*http.Response, *http.Request)
}

func (r *Response) BodyJSON(b string) *Response {
	r.bodyJSON = b
	return r
}

func (r *Response) BodyText(b string) *Response {
	r.body = b
	return r
}

func (r *Response) Cookies(cookies map[string]string) *Response {
	r.cookies = cookies
	return r
}

func (r *Response) CookiePresent(cookieName string) *Response {
	r.cookiesPresent = append(r.cookiesPresent, cookieName)
	return r
}

func (r *Response) Headers(headers map[string]string) *Response {
	r.headers = headers
	return r
}

func (r *Response) Status(s int) *Response {
	r.status = s
	return r
}

func (r *Response) Assert(fn func(*http.Response, *http.Request)) *Response {
	r.assert = fn
	return r.apiTest.response
}

func (r *Response) JSONPath(expression string, assert func(interface{})) *Response {
	r.jsonPathExpression = expression
	r.jsonPathAssert = assert
	return r.apiTest.response
}

func (r *Response) End() {
	r.apiTest.Run()
}

func (a *APITest) Run() {
	res, req := a.runTest()
	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)
	a.assertJSONPath(res)
	if a.response.assert != nil {
		a.response.assert(res.Result(), req)
	}
}

func (a *APITest) runTest() (*httptest.ResponseRecorder, *http.Request) {
	req := a.buildRequestFromTestCase()
	res := httptest.NewRecorder()
	a.handler.ServeHTTP(res, req)
	return res, req
}

func (a *APITest) buildRequestFromTestCase() *http.Request {
	var body string
	var contentType string
	if a.request.bodyJSON != "" {
		body = a.request.bodyJSON
		contentType = "application/json"
	} else if a.request.body != "" {
		body = a.request.body
		contentType = "text/plain"
	}

	req, _ := http.NewRequest(a.request.method, a.request.url, bytes.NewBufferString(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if a.request.query != nil {
		query := req.URL.Query()
		for k, v := range a.request.query {
			query.Add(k, v)
		}
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

func (a *APITest) assertResponse(res *httptest.ResponseRecorder) {
	if a.response.status != 0 {
		assert.Equal(a.t, a.response.status, res.Code, a.name)
	}
	if a.response.bodyJSON != "" {
		assert.JSONEq(a.t, a.response.bodyJSON, res.Body.String(), a.name)
		return
	}

	if a.response.body != "" {
		assert.Equal(a.t, a.response.body, res.Body.String(), a.name)
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
