package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
)

// Request is the user defined request that will be invoked on the handler under test
type Request struct {
	interceptor       Intercept
	method            string
	url               string
	body              string
	query             map[string][]string
	queryCollection   map[string][]string
	headers           map[string][]string
	formData          map[string][]string
	multipartBody     *bytes.Buffer
	multipart         *multipart.Writer
	cookies           []*Cookie
	basicAuthSet      bool
	basicAuthUsername string
	basicAuthPassword string
	context           context.Context
	apiTest           *APITest
	err               error
}

// Intercept will be called before the request is made. Updates to the request will be reflected in the test
type Intercept func(*http.Request)

// URL is a builder method for setting the url of the request
func (r *Request) URL(url string) *Request {
	r.url = url
	return r
}

// URLf is a builder method for setting the url of the request and supports a formatter
func (r *Request) URLf(format string, args ...any) *Request {
	r.url = fmt.Sprintf(format, args...)
	return r
}

// Body is a builder method to set the request body
func (r *Request) Body(b string) *Request {
	r.body = b
	return r
}

// Bodyf sets the request body and supports a formatter
func (r *Request) Bodyf(format string, args ...any) *Request {
	r.body = fmt.Sprintf(format, args...)
	return r
}

// BodyFromFile is a builder method to set the request body
func (r *Request) BodyFromFile(f string) *Request {
	b, err := os.ReadFile(f)
	if err != nil {
		r.fail(err)
		return r
	}
	r.body = string(b)
	return r
}

// JSON is a convenience method for setting the request body and content type header as "application/json".
// If v is not a string or []byte it will marshall the provided variable as json
func (r *Request) JSON(v any) *Request {
	switch x := v.(type) {
	case string:
		r.body = x
	case []byte:
		r.body = string(x)
	default:
		asJSON, err := json.Marshal(x)
		if err != nil {
			r.fail(err)
			return r
		}
		r.body = string(asJSON)
	}
	r.ContentType("application/json")
	return r
}

// JSONFromFile is a convenience method for setting the request body and content type header as "application/json"
func (r *Request) JSONFromFile(f string) *Request {
	r.BodyFromFile(f)
	r.ContentType("application/json")
	return r
}

// GraphQLQuery is a convenience method for building a graphql POST request
func (r *Request) GraphQLQuery(query string, variables ...map[string]any) *Request {
	q := GraphQLRequestBody{
		Query: query,
	}

	if len(variables) > 0 {
		q.Variables = variables[0]
	}

	return r.GraphQLRequest(q)
}

// GraphQLRequest builds a graphql POST request
func (r *Request) GraphQLRequest(body GraphQLRequestBody) *Request {
	r.ContentType("application/json")

	data, err := json.Marshal(body)
	if err != nil {
		r.fail(err)
		return r
	}

	r.body = string(data)

	return r
}

// GraphQLRequestBody represents the POST request body as per the GraphQL spec
type GraphQLRequestBody struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
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

// ContentType is a builder method to set the Content-Type header of the request
func (r *Request) ContentType(contentType string) *Request {
	normalizedKey := textproto.CanonicalMIMEHeaderKey("Content-Type")
	r.headers[normalizedKey] = []string{contentType}
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
	r.basicAuthSet = true
	r.basicAuthUsername = username
	r.basicAuthPassword = password
	return r
}

// WithContext is a builder method to set a context on the request
func (r *Request) WithContext(ctx context.Context) *Request {
	r.context = ctx
	return r
}

// FormData is a builder method to set the body form data
// Also sets the content type of the request to application/x-www-form-urlencoded
func (r *Request) FormData(name string, values ...string) *Request {
	defer r.checkCombineFormDataWithMultipart()

	r.ContentType("application/x-www-form-urlencoded")
	r.formData[name] = append(r.formData[name], values...)
	return r
}

// MultipartFormData is a builder method to set the field in multipart form data
// Also sets the content type of the request to multipart/form-data
func (r *Request) MultipartFormData(name string, values ...string) *Request {
	defer r.checkCombineFormDataWithMultipart()

	r.setMultipartWriter()

	for _, value := range values {
		if err := r.multipart.WriteField(name, value); err != nil {
			r.fail(err)
			return r
		}
	}

	return r
}

// MultipartFile is a builder method to set the file in multipart form data
// Also sets the content type of the request to multipart/form-data
func (r *Request) MultipartFile(name string, ff ...string) *Request {
	defer r.checkCombineFormDataWithMultipart()

	r.setMultipartWriter()

	for _, f := range ff {
		func() {
			file, err := r.apiTest.fileSystem.Open(f)
			if err != nil {
				r.fail(err)
				return
			}
			defer file.Close()

			part, err := r.multipart.CreateFormFile(name, filepath.Base(f))
			if err != nil {
				r.fail(err)
				return
			}

			if _, err = io.Copy(part, file); err != nil {
				r.fail(err)
				return
			}
		}()
	}

	return r
}

func (r *Request) setMultipartWriter() {
	if r.multipart == nil {
		r.multipartBody = &bytes.Buffer{}
		r.multipart = multipart.NewWriter(r.multipartBody)
	}
}

func (r *Request) checkCombineFormDataWithMultipart() {
	if r.multipart != nil && len(r.formData) > 0 {
		r.fail(errors.New("FormData (application/x-www-form-urlencoded) and MultiPartFormData(multipart/form-data) cannot be combined"))
	}
}

// Expect marks the request spec as complete and following code will define the expected response
func (r *Request) Expect(t TestingT) *Response {
	r.apiTest.t = t
	if r.err != nil {
		t.Fatal(r.err)
	}
	return r.apiTest.response
}

// fail records an error raised while building the request. A TestingT is only
// available once Expect has been called, so until then the first error is kept
// and reported by Expect.
func (r *Request) fail(err error) {
	if r.apiTest.t != nil {
		r.apiTest.t.Fatal(err)
		return
	}
	if r.err == nil {
		r.err = err
	}
}

func (a *APITest) buildRequest() *http.Request {
	if a.httpRequest != nil {
		return a.httpRequest
	}

	if len(a.request.formData) > 0 {
		form := url.Values{}
		for k := range a.request.formData {
			for _, value := range a.request.formData[k] {
				form.Add(k, value)
			}
		}
		a.request.Body(form.Encode())
	}

	if a.request.multipart != nil {
		err := a.request.multipart.Close()
		if err != nil {
			a.request.apiTest.t.Fatal(err)
		}

		a.request.Header("Content-Type", a.request.multipart.FormDataContentType())
		a.request.Body(a.request.multipartBody.String())
	}

	req, _ := http.NewRequest(a.request.method, a.request.url, bytes.NewBufferString(a.request.body))
	if a.request.context != nil {
		req = req.WithContext(a.request.context)
	}

	req.URL.RawQuery = formatQuery(a.request)
	req.Host = SystemUnderTestDefaultName
	if len(a.host) > 0 {
		req.Host = a.host
	}

	if a.networkingEnabled {
		req.Host = req.URL.Host
	}

	for k, v := range a.request.headers {
		for _, headerValue := range v {
			req.Header.Add(k, headerValue)
		}
	}

	for _, cookie := range a.request.cookies {
		req.AddCookie(cookie.ToHttpCookie())
	}

	if a.request.basicAuthSet {
		req.SetBasicAuth(a.request.basicAuthUsername, a.request.basicAuthPassword)
	}

	return req
}

func formatQuery(request *Request) string {
	out := url.Values{}

	for key, values := range request.queryCollection {
		for _, value := range values {
			out.Add(key, value)
		}
	}

	for key, values := range request.query {
		for _, value := range values {
			out.Add(key, value)
		}
	}

	return out.Encode()
}
