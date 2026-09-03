package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
)

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
					return fmt.Errorf("failed to parse regexp for header %s with value %s", key, value)
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

var basicAuthMatcher = func(req *http.Request, spec *MockRequest) error {
	if spec.basicAuthUsername == "" {
		return nil
	}

	username, password, ok := req.BasicAuth()
	if !ok {
		return errors.New("request did not contain valid HTTP Basic Authentication string")
	}

	if spec.basicAuthUsername != username {
		return fmt.Errorf("basic auth request username '%s' did not match mock username '%s'",
			username, spec.basicAuthUsername)
	}

	if spec.basicAuthPassword != password {
		return fmt.Errorf("basic auth request password '%s' did not match mock password '%s'",
			password, spec.basicAuthPassword)
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
	receivedQueryParams := req.URL.Query()

	for key, values := range mockQueryParams {
		if _, ok := receivedQueryParams[key]; !ok {
			return fmt.Errorf("not all of received query params %s matched expected mock query params %s", receivedQueryParams, mockQueryParams)
		}

		found, pattern, err := matchedValueCount(values, receivedQueryParams[key])
		if err != nil {
			return fmt.Errorf("failed to parse regexp for query param %s with value %s", key, pattern)
		}

		if found != len(values) {
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

var formDataMatcher = func(req *http.Request, spec *MockRequest) error {
	mockFormData := spec.formData
	if len(mockFormData) == 0 {
		return nil
	}

	r := copyHttpRequest(req)
	if err := r.ParseForm(); err != nil {
		return errors.New("unable to parse form data")
	}

	receivedFormData := r.PostForm

	for key, values := range mockFormData {
		if _, ok := receivedFormData[key]; !ok {
			return fmt.Errorf("not all of received form data values %s matched expected mock form data values %s",
				receivedFormData, mockFormData)
		}

		found, pattern, err := matchedValueCount(values, receivedFormData[key])
		if err != nil {
			return fmt.Errorf("failed to parse regexp for form data %s with value %s", key, pattern)
		}

		if found != len(values) {
			return fmt.Errorf("not all of received form data values %s matched expected mock form data values %s", receivedFormData, mockFormData)
		}
	}
	return nil
}

var formDataPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	if len(spec.formDataPresent) > 0 {
		r := copyHttpRequest(req)
		if err := r.ParseForm(); err != nil {
			return errors.New("unable to parse form data")
		}

		receivedFormData := r.PostForm

		for _, key := range spec.formDataPresent {
			if _, ok := receivedFormData[key]; !ok {
				return fmt.Errorf("expected form data key %s not received", key)
			}
		}
	}
	return nil
}

var formDataNotPresentMatcher = func(req *http.Request, spec *MockRequest) error {
	if len(spec.formDataNotPresent) > 0 {
		r := copyHttpRequest(req)
		if err := r.ParseForm(); err != nil {
			return errors.New("unable to parse form data")
		}

		receivedFormData := r.PostForm

		for _, key := range spec.formDataNotPresent {
			if _, ok := receivedFormData[key]; ok {
				return fmt.Errorf("did not expect a form data key %s", key)
			}
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

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("expected a body but received none")
	}

	// replace body so it can be read again
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Perform exact string match
	bodyStr := string(body)
	if bodyStr == mockBody {
		return nil
	}

	// Perform JSON match
	var reqJSON any
	reqJSONErr := json.Unmarshal(body, &reqJSON)

	var matchJSON any
	specJSONErr := json.Unmarshal([]byte(mockBody), &matchJSON)

	isJSON := reqJSONErr == nil && specJSONErr == nil
	if isJSON && reflect.DeepEqual(reqJSON, matchJSON) {
		return nil
	}

	if isJSON {
		return fmt.Errorf("received body did not match expected mock body\n%s", diff(matchJSON, reqJSON))
	}

	return fmt.Errorf("received body did not match expected mock body\n%s", diff(mockBody, bodyStr))
}

var bodyRegexpMatcher = func(req *http.Request, spec *MockRequest) error {
	expression := spec.bodyRegexp

	if len(expression) == 0 {
		return nil
	}

	if req.Body == nil {
		return errors.New("expected a body but received none")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("expected a body but received none")
	}

	// replace body so it can be read again
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Perform regexp match
	bodyStr := string(body)
	match, _ := regexp.MatchString(expression, bodyStr)
	if match {
		return nil
	}

	return fmt.Errorf("received body did not match expected mock body\n%s", diff(expression, bodyStr))
}

// matchedValueCount counts the (received, expected) pairs in which the expected value, treated as a
// regexp, matches the received value. Callers compare the count with len(expected), which is the
// matching rule the query and form data matchers have always used. If an expected value is not a
// valid regexp it is returned along with the error.
func matchedValueCount(expected, received []string) (found int, invalidPattern string, err error) {
	for _, field := range received {
		for _, value := range expected {
			match, err := regexp.MatchString(value, field)
			if err != nil {
				return 0, value, err
			}
			if match {
				found++
			}
		}
	}
	return found, "", nil
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
	basicAuthMatcher,
	headerPresentMatcher,
	headerNotPresentMatcher,
	queryParamMatcher,
	queryPresentMatcher,
	queryNotPresentMatcher,
	formDataMatcher,
	formDataPresentMatcher,
	formDataNotPresentMatcher,
	bodyMatcher,
	bodyRegexpMatcher,
	cookieMatcher,
	cookiePresentMatcher,
	cookieNotPresentMatcher,
}
