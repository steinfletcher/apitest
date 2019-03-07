package apitest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestMocks_QueryPresent(t *testing.T) {
	tests := []struct {
		requestUrl    string
		queryParam    string
		expectedError error
	}{
		{"http://test.com/v1/path?a=1", "a", nil},
		{"http://test.com/v1/path", "a", errors.New("expected query param a not received")},
		{"http://test.com/v1/path?c=1", "b", errors.New("expected query param b not received")},
		{"http://test.com/v2/path?b=2&a=1", "a", nil},
	}
	for _, test := range tests {
		t.Run(test.requestUrl, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			mockRequest := NewMock().Get(test.requestUrl).QueryPresent(test.queryParam)
			matchError := queryPresentMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocks_NewUnmatchedMockError_Empty(t *testing.T) {
	mockError := newUnmatchedMockError()

	assert.NotNil(t, mockError)
	assert.Len(t, mockError.errors, 0)
}

func TestMocks_NewEmptyUnmatchedMockError_ExpectedErrorsString(t *testing.T) {
	mockError := newUnmatchedMockError().
		addErrors(1, errors.New("a boo boo has occurred")).
		addErrors(2, errors.New("tom drank too much beer"))

	assert.NotNil(t, mockError)
	assert.Len(t, mockError.errors, 2)
	assert.Equal(t,
		"received request did not match any mocks\n\nMock 1 mismatches:\n• a boo boo has occurred\n\nMock 2 mismatches:\n• tom drank too much beer\n\n",
		mockError.Error())
}

func TestMocks_HostMatcher(t *testing.T) {
	tests := map[string]struct {
		request       *http.Request
		mockUrl       string
		expectedError error
	}{
		"matching": {
			request:       httptest.NewRequest(http.MethodGet, "http://test.com", nil),
			mockUrl:       "https://test.com",
			expectedError: nil,
		},
		"not matching": {
			request:       httptest.NewRequest(http.MethodGet, "https://test.com", nil),
			mockUrl:       "https://testa.com",
			expectedError: errors.New("received host test.com did not match mock host testa.com"),
		},
		"no expected host": {
			request:       httptest.NewRequest(http.MethodGet, "https://test.com", nil),
			mockUrl:       "",
			expectedError: nil,
		},
		"matching using URL host": {
			request: &http.Request{URL: &url.URL{
				Host: "test.com",
				Path: "/",
			}},
			mockUrl:       "https://test.com",
			expectedError: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matchError := hostMatcher(test.request, NewMock().Get(test.mockUrl))
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocks_HeaderMatcher(t *testing.T) {
	tests := []struct {
		requestHeaders     map[string]string
		headerToMatchKey   string
		headerToMatchValue string
		expectedError      error
	}{
		{map[string]string{"B": "5", "A": "123"}, "A", "123", nil},
		{map[string]string{"A": "123"}, "C", "3", errors.New("not all of received headers map[A:[123]] matched expected mock headers map[C:[3]]")},
		{map[string]string{}, "", "", nil},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.headerToMatchKey, test.headerToMatchValue), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/assert", nil)
			for k, v := range test.requestHeaders {
				req.Header.Set(k, v)
			}
			mockRequest := NewMock().Get("/assert")
			if test.headerToMatchKey != "" {
				mockRequest.Header(test.headerToMatchKey, test.headerToMatchValue)
			}
			matchError := headerMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocks_MockRequest_Header_WorksWithHeaders(t *testing.T) {
	mock := NewMock().
		Get("/path").
		Header("A", "12345").
		Headers(map[string]string{"B": "67890"})
	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	req.Header.Set("A", "12345")
	req.Header.Set("B", "67890")

	matchError := headerMatcher(req, mock)

	assert.Nil(t, matchError)
}

func TestMocks_QueryMatcher(t *testing.T) {
	tests := []struct {
		requestUrl    string
		queryToMatch  map[string]string
		expectedError error
	}{
		{"http://test.com/v1/path?a=1", map[string]string{"a": "1"}, nil},
		{"http://test.com/v1/path", map[string]string{"a": "1"}, errors.New("not all of received query params map[] matched expected mock query params map[a:[1]]")},
		{"http://test.com/v2/path?a=1", map[string]string{"b": "1"}, errors.New("not all of received query params map[a:[1]] matched expected mock query params map[b:[1]]")},
		{"http://test.com/v2/path?b=2&a=1", map[string]string{"a": "1"}, nil},
	}
	for _, test := range tests {
		t.Run(test.requestUrl, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			mockRequest := NewMock().Get(test.requestUrl).QueryParams(test.queryToMatch)
			matchError := queryParamMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocks_QueryParams_DoesNotOverwriteQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.com/v2/path?b=2&a=1", nil)
	mockRequest := NewMock().
		Get("http://test.com").
		Query("b", "2").
		QueryParams(map[string]string{"a": "1"})

	matchError := queryParamMatcher(req, mockRequest)

	assert.Equal(t, 2, len(mockRequest.query))
	assert.Nil(t, matchError)
}

func TestMocks_SchemeMatcher(t *testing.T) {
	tests := []struct {
		requestUrl    string
		mockUrl       string
		expectedError error
	}{
		{"http://test.com", "https://test.com", errors.New("received scheme http did not match mock scheme https")},
		{"https://test.com", "https://test.com", nil},
		{"https://test.com", "test.com", nil},
		{"localhost:80", "localhost:80", nil},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.requestUrl, test.mockUrl), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			matchError := schemeMatcher(req, NewMock().Get(test.mockUrl))
			if !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("mockUrl='%s' requestUrl='%s' actual=%v shouldMatch=%v",
					test.mockUrl, test.requestUrl, matchError, test.expectedError)
			}
		})
	}
}

func TestMocks_BodyMatcher(t *testing.T) {
	tests := []struct {
		requestBody   string
		matchBody     string
		expectedError error
	}{
		{`{"a": 1}`, "", nil},
		{``, `{"a":1}`, errors.New("expected a body but received none")},
		{"golang\n", "go[lang]?", nil},
		{"golang\n", "go[lang]?", nil},
		{"golang", "goat", errors.New("received body golang did not match expected mock body goat")},
		{"go\n", "go[lang]?", nil},
		{`{"a":"12345"}\n`, `{"a":"12345"}`, nil},
		{`{"a":"12345"}`, `{"b":"12345"}`, errors.New(`received body {"a":"12345"} did not match expected mock body {"b":"12345"}`)},
		{`{"x":"12345"}`, `{"x":"12345"}`, nil},
		{`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`,
			`{"b":[{"key":"c","value":"result"}],"a":12345}`, nil},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("body=%v", test.matchBody), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(test.requestBody))
			matchError := bodyMatcher(req, NewMock().Get("/path").Body(test.matchBody))
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocks_PathMatcher(t *testing.T) {
	tests := []struct {
		requestUrl    string
		pathToMatch   string
		expectedError error
	}{
		{"http://test.com/v1/path", "/v1/path", nil},
		{"http://test.com/v1/path", "/v1/not", errors.New("received path /v1/path did not match mock path /v1/not")},
		{"http://test.com/v1/path", "", nil},
		{"http://test.com", "", nil},
		{"http://test.com/v2/path", "/v2/.+th", nil},
	}
	for _, test := range tests {
		t.Run(test.pathToMatch, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			matchError := pathMatcher(req, NewMock().Get(test.pathToMatch))
			if matchError != nil && !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("methodToMatch='%s' requestUrl='%s' shouldMatch=%v",
					test.pathToMatch, test.requestUrl, matchError)
			}
		})
	}
}

func TestMocks_PanicsIfUrlInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected to panic")
		}
	}()

	NewMock().Get("http:// blah")
}

func TestMocks_Matches(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	getPreferences := NewMock().
		Get("/preferences/12345").
		RespondWith().
		Body(`{"is_contactable": true}`).
		Status(http.StatusOK).
		End()
	getUser := NewMock().
		Get("/user/1234").
		RespondWith().
		Status(http.StatusOK).
		Body(`{"name": "jon", "id": "1234"}`).
		End()

	mockResponse, matchErrors := matches(req, []*Mock{getUser, getPreferences})

	assert.Nil(t, matchErrors)
	assert.NotNil(t, mockResponse)
	assert.Equal(t, `{"is_contactable": true}`, mockResponse.body)
}

func TestMocks_Matches_NilIfNoMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	mockResponse, matchErrors := matches(req, []*Mock{})

	if mockResponse != nil {
		t.Fatal("Expected nil")
	}

	assert.NotNil(t, matchErrors)
	assert.Equal(t, newUnmatchedMockError(), matchErrors)
}

func TestMocks_UnmatchedMockErrorOrderedMockKeys(t *testing.T) {
	unmatchedMockError := newUnmatchedMockError().
		addErrors(3, errors.New("oh no")).
		addErrors(1, errors.New("oh shoot")).
		addErrors(4, errors.New("gah"))

	assert.Equal(t,
		"received request did not match any mocks\n\nMock 1 mismatches:\n• oh shoot\n\nMock 3 mismatches:\n• oh no\n\nMock 4 mismatches:\n• gah\n\n",
		unmatchedMockError.Error())
}

func TestMocks_Matches_ErrorsMatchUnmatchedMocks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	mockResponse, matchErrors := matches(req,
		[]*Mock{
			NewMock().
				Get("/preferences/123456").
				RespondWith().
				End()})

	if mockResponse != nil {
		t.Fatal("Expected nil")
	}

	assert.NotNil(t, matchErrors)
	assert.Equal(t, "received request did not match any mocks\n\nMock 1 mismatches:\n• received path /preferences/12345 did not match mock path /preferences/123456\n\n",
		matchErrors.Error())
}

func TestMocks_MethodMatcher(t *testing.T) {
	tests := []struct {
		requestMethod string
		methodToMatch string
		expectedError error
	}{
		{http.MethodGet, http.MethodGet, nil},
		{http.MethodPost, http.MethodPost, nil},
		{http.MethodDelete, "", nil},
		{http.MethodPut, http.MethodGet, errors.New("received method PUT did not match mock method GET")},
		{"", http.MethodGet, nil},
		{"", "", nil},
		{http.MethodOptions, http.MethodGet, errors.New("received method OPTIONS did not match mock method GET")},
	}
	for _, test := range tests {
		t.Run(test.requestMethod, func(t *testing.T) {
			req := httptest.NewRequest(test.requestMethod, "/path", nil)
			matchError := methodMatcher(req, NewMock().Method(test.methodToMatch))
			if !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("methodToMatch='%s' requestMethod='%s' actual=%v shouldMatch=%v",
					test.methodToMatch, test.requestMethod, matchError, test.expectedError)
			}
		})
	}
}

func TestMocks_Request_SetsTheMethod(t *testing.T) {
	tests := []struct {
		expectedMethod string
		methodSetter   func(m *Mock)
	}{
		{http.MethodGet, func(m *Mock) { m.Get("/") }},
		{http.MethodPost, func(m *Mock) { m.Post("/") }},
		{http.MethodPut, func(m *Mock) { m.Put("/") }},
		{http.MethodDelete, func(m *Mock) { m.Delete("/") }},
		{http.MethodPatch, func(m *Mock) { m.Patch("/") }},
	}
	for _, test := range tests {
		t.Run(test.expectedMethod, func(t *testing.T) {
			mock := NewMock()
			test.methodSetter(mock)
			assert.Equal(t, test.expectedMethod, mock.request.method)
		})
	}
}

func TestMocks_Response_SetsTextPlainIfNoContentTypeSet(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body("abcdef")

	response := buildResponseFromMock(mockResponse)

	bytes, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, string(bytes), "abcdef")
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))
}

func TestMocks_Response_SetsTheBodyAsJSON(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body(`{"a": 123}`)

	response := buildResponseFromMock(mockResponse)

	bytes, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, string(bytes), `{"a": 123}`)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
}

func TestMocks_Response_SetsTheBodyAsOther(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body(`<html>123</html>`).
		Header("Content-Type", "text/html")

	response := buildResponseFromMock(mockResponse)

	bytes, _ := ioutil.ReadAll(response.Body)
	assert.Equal(t, string(bytes), `<html>123</html>`)
	assert.Equal(t, "text/html", response.Header.Get("Content-Type"))
}

func TestMocks_Response_Headers_WithNormalizedKeys(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Header("a", "1").
		Headers(map[string]string{"B": "2"}).
		Header("c", "3")

	response := buildResponseFromMock(mockResponse)

	assert.Equal(t, http.Header(map[string][]string{"A": {"1"}, "B": {"2"}, "C": {"3"}}), response.Header)
}

func TestMocks_Response_Cookies(t *testing.T) {
	mockResponse := NewMock().
		Get("test").
		RespondWith().
		Cookie("A", "1").
		Cookies(NewCookie("B").Value("2")).
		Cookie("C", "3")

	response := buildResponseFromMock(mockResponse)

	assert.Equal(t, []*http.Cookie{
		{Name: "A", Value: "1", Raw: "A=1"},
		{Name: "B", Value: "2", Raw: "B=2"},
		{Name: "C", Value: "3", Raw: "C=3"},
	}, response.Cookies())
}

func TestMocks_ApiTest_WithMocks(t *testing.T) {
	tests := []struct {
		name    string
		httpCli *http.Client
	}{
		{name: "custom http cli", httpCli: customCli},
		{name: "default http cli", httpCli: http.DefaultClient},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getUser := NewMock().
				Get("/user").
				RespondWith().
				Body(`{"name": "jon", "id": "1234"}`).
				Status(http.StatusOK).
				End()

			getPreferences := NewMock().
				Get("/preferences").
				RespondWith().
				Body(`{"is_contactable": false}`).
				Status(http.StatusOK).
				End()

			New().
				Debug().
				HttpClient(test.httpCli).
				Mocks(getUser, getPreferences).
				Handler(getUserHandler(NewHttpGet(test.httpCli))).
				Get("/user").
				Expect(t).
				Status(http.StatusOK).
				Body(`{"name": "jon", "is_contactable": false}`).
				End()
		})
	}
}

func TestMocks_ApiTest_SupportsObservingMocks(t *testing.T) {
	var observedMocks []*mockInteraction

	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	getPreferences := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("2").
		End()

	New().
		ObserveMocks(func(res *http.Response, req *http.Request, a *APITest) {
			if res == nil || req == nil || a == nil {
				t.Fatal("expected request and response to be defined")
			}
			observedMocks = append(observedMocks, &mockInteraction{response: res, request: req})
		}).
		Mocks(getUser, getPreferences).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes1 := getUserData()
			bytes2 := getUserData()
			bytes3 := getUserData()

			w.Write(bytes1)
			w.Write(bytes2)
			w.Write(bytes3)
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Body(`112`).
		End()

	assert.Equal(t, 3, len(observedMocks))
}

func TestMocks_ApiTest_SupportsMultipleMocks(t *testing.T) {
	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	getPreferences := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("2").
		End()

	New().
		Mocks(getUser, getPreferences).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes1 := getUserData()
			bytes2 := getUserData()
			bytes3 := getUserData()

			w.Write(bytes1)
			w.Write(bytes2)
			w.Write(bytes3)
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Body(`112`).
		End()
}

func getUserData() []byte {
	res, err := http.Get("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return bytes
}

func getUserHandler(get HttpGet) *http.ServeMux {
	handler := http.NewServeMux()
	handler.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		var user User
		get("/user", &user)

		var contactPreferences ContactPreferences
		get("/preferences", &contactPreferences)

		response := UserResponse{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}
		bytes, _ := json.Marshal(response)
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	})
	return handler
}

type User struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ContactPreferences struct {
	IsContactable bool `json:"is_contactable"`
}

type UserResponse struct {
	Name          string `json:"name"`
	IsContactable bool   `json:"is_contactable"`
}

var customCli = &http.Client{
	Transport: &http.Transport{},
}

type HttpGet func(path string, response interface{})

func NewHttpGet(cli *http.Client) HttpGet {
	return func(path string, response interface{}) {
		res, err := cli.Get(fmt.Sprintf("http://localhost:8080%s", path))
		if err != nil {
			panic(err)
		}

		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(bytes, response)
		if err != nil {
			panic(err)
		}
	}
}
