package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"
	"time"
)

// Transport wraps components used to observe and manipulate the real request and response objects
type Transport struct {
	debugEnabled             bool
	mockResponseDelayEnabled bool
	mocks                    []*Mock
	nativeTransport          http.RoundTripper
	httpClient               *http.Client
	observers                []Observe
	apiTest                  *APITest
}

func newTransport(
	mocks []*Mock,
	httpClient *http.Client,
	debugEnabled bool,
	mockResponseDelayEnabled bool,
	observers []Observe,
	apiTest *APITest) *Transport {

	t := &Transport{
		mocks:                    mocks,
		httpClient:               httpClient,
		debugEnabled:             debugEnabled,
		mockResponseDelayEnabled: mockResponseDelayEnabled,
		observers:                observers,
		apiTest:                  apiTest,
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

// Error implementation of in-built error human readable string function
func (u *unmatchedMockError) Error() string {
	var strBuilder strings.Builder
	strBuilder.WriteString("received request did not match any mocks\n\n")
	for _, mockNumber := range u.orderedMockKeys() {
		fmt.Fprintf(&strBuilder, "Mock %d mismatches:\n", mockNumber)
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

	if len(r.observers) > 0 {
		defer func() {
			for _, observe := range r.observers {
				observe(mockResponse, req, r.apiTest)
			}
		}()
	}

	matchedResponse, matchErrors := matches(req, r.mocks)
	if matchErrors == nil {
		res := buildResponseFromMock(matchedResponse)
		res.Request = req

		if matchedResponse.timeout {
			return nil, timeoutError{}
		}

		if r.mockResponseDelayEnabled && matchedResponse.fixedDelayMillis > 0 {
			time.Sleep(time.Duration(matchedResponse.fixedDelayMillis) * time.Millisecond)
		}

		return res, nil
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

// Hijack replace the transport implementation of the interaction under test in order to observe, mock and inject expectations
func (r *Transport) Hijack() {
	if r.httpClient != nil {
		r.httpClient.Transport = r
		return
	}
	http.DefaultTransport = r
}

// Reset replace the hijacked transport implementation of the interaction under test to the original implementation
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

	mockResponse.mu.RLock()
	defer mockResponse.mu.RUnlock()

	// copy the headers so that the response does not share (and mutate) the mock's map
	headers := make(http.Header, len(mockResponse.headers)+2)
	for key, values := range mockResponse.headers {
		headers[key] = append([]string(nil), values...)
	}

	// if the content type isn't set and the body contains json, set content type as json
	if len(mockResponse.body) > 0 {
		if contentTypeHeader := headers["Content-Type"]; len(contentTypeHeader) == 0 {
			if json.Valid([]byte(mockResponse.body)) {
				headers.Set("Content-Type", "application/json")
			} else {
				headers.Set("Content-Type", "text/plain")
			}
		} else {
			headers.Set("Content-Type", contentTypeHeader[0])
		}
	}

	for _, cookie := range mockResponse.cookies {
		if v := cookie.ToHttpCookie().String(); v != "" {
			headers.Add("Set-Cookie", v)
		}
	}

	return &http.Response{
		Body:          io.NopCloser(strings.NewReader(mockResponse.body)),
		Header:        headers,
		StatusCode:    mockResponse.statusCode,
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(mockResponse.body)),
	}
}

func matches(req *http.Request, mocks []*Mock) (*MockResponse, error) {
	mockError := newUnmatchedMockError()
	for mockNumber, mock := range mocks {
		mock.m.Lock() // lock is for isUsed when matches is called concurrently by RoundTripper
		if mock.isUsed && !mock.anyTimesSet {
			mock.m.Unlock()
			continue
		}

		errs := mock.Matches(req)
		if len(errs) == 0 {
			mock.isUsed = true
			mock.m.Unlock()
			return mock.response, nil
		}

		mockError = mockError.addErrors(mockNumber+1, errs...)
		mock.m.Unlock()
	}

	return nil, mockError
}

type timeoutError struct{}
