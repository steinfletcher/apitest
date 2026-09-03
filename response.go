package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
)

// Response is the user defined expected response from the application under test
type Response struct {
	status            int
	body              string
	headers           map[string][]string
	headersPresent    []string
	headersNotPresent []string
	cookies           []*Cookie
	cookiesPresent    []string
	cookiesNotPresent []string
	apiTest           *APITest
	assert            []Assert
}

// Assert is a user defined custom assertion function
type Assert func(*http.Response, *http.Request) error

// Body is the expected response body
func (r *Response) Body(b string) *Response {
	r.body = b
	return r
}

// Bodyf is the expected response body that supports a formatter
func (r *Response) Bodyf(format string, args ...any) *Response {
	r.body = fmt.Sprintf(format, args...)
	return r
}

// BodyFromFile reads the given file and uses the content as the expected response body
func (r *Response) BodyFromFile(f string) *Response {
	b, err := os.ReadFile(f)
	if err != nil {
		r.apiTest.t.Fatal(err)
	}
	r.body = string(b)
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
	normalizedName := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedName] = append(r.headers[normalizedName], value)
	return r
}

// HeaderPresent is a builder method to set the request headers that should be present in the response
func (r *Response) HeaderPresent(name string) *Response {
	normalizedName := textproto.CanonicalMIMEHeaderKey(name)
	r.headersPresent = append(r.headersPresent, normalizedName)
	return r
}

// HeaderNotPresent is a builder method to set the request headers that should not be present in the response
func (r *Response) HeaderNotPresent(name string) *Response {
	normalizedName := textproto.CanonicalMIMEHeaderKey(name)
	r.headersNotPresent = append(r.headersNotPresent, normalizedName)
	return r
}

// Headers is a builder method to set the request headers
func (r *Response) Headers(headers map[string]string) *Response {
	for name, value := range headers {
		normalizedName := textproto.CanonicalMIMEHeaderKey(name)
		r.headers[normalizedName] = append(r.headers[normalizedName], value)
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
	return r
}

// Result provides the final result
type Result struct {
	Response       *http.Response
	unmatchedMocks []UnmatchedMock
}

// UnmatchedMocks returns any mocks that were not used, e.g. there was not a matching http Request for the mock
func (r Result) UnmatchedMocks() []UnmatchedMock {
	return r.unmatchedMocks
}

// JSON unmarshal the result response body to a valid struct
func (r Result) JSON(t any) {
	data, err := io.ReadAll(r.Response.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, t)
	if err != nil {
		panic(err)
	}
}
