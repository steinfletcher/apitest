package apitest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMocks_QueryPresent(t *testing.T) {
	tests := []struct {
		requestUrl string
		queryParam string
		isPresent  bool
	}{
		{"http://test.com/v1/path?a=1", "a", true},
		{"http://test.com/v1/path", "a", false},
		{"http://test.com/v1/path?b=1", "a", false},
		{"http://test.com/v2/path?b=2&a=1", "a", true},
	}
	for _, test := range tests {
		t.Run(test.requestUrl, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			mockRequest := NewMock().Get(test.requestUrl).QueryPresent(test.queryParam)
			isPresent := queryPresentMatcher(req, mockRequest)
			assertEqual(t, test.isPresent, isPresent)
		})
	}
}

func TestMocks_HostMatcher(t *testing.T) {
	tests := []struct {
		requestUrl  string
		mockUrl     string
		shouldMatch bool
	}{
		{"http://test.com", "https://test.com", true},
		{"https://test.com", "https://testa.com", false},
		{"https://test.com", "", true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.requestUrl, test.mockUrl), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			matches := hostMatcher(req, NewMock().Get(test.mockUrl))
			assertEqual(t, test.shouldMatch, matches)
		})
	}
}

func TestMocks_HeaderMatcher(t *testing.T) {
	tests := []struct {
		requestHeaders     map[string]string
		headerToMatchKey   string
		headerToMatchValue string
		shouldMatch        bool
	}{
		{map[string]string{"B": "5", "A": "123"}, "A", "123", true},
		{map[string]string{"A": "123"}, "C", "3", false},
		{map[string]string{}, "", "", true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.headerToMatchKey, test.headerToMatchValue), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range test.requestHeaders {
				req.Header.Set(k, v)
			}
			mockRequest := NewMock().Get("/test")
			if test.headerToMatchKey != "" {
				mockRequest.Header(test.headerToMatchKey, test.headerToMatchValue)
			}
			matches := headerMatcher(req, mockRequest)
			assertEqual(t, test.shouldMatch, matches)
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

	matches := headerMatcher(req, mock)

	assertEqual(t, true, matches)
}

func TestMocks_QueryMatcher(t *testing.T) {
	tests := []struct {
		requestUrl   string
		queryToMatch map[string]string
		shouldMatch  bool
	}{
		{"http://test.com/v1/path?a=1", map[string]string{"a": "1"}, true},
		{"http://test.com/v1/path", map[string]string{"a": "1"}, false},
		{"http://test.com/v2/path?a=1", map[string]string{"b": "1"}, false},
		{"http://test.com/v2/path?b=2&a=1", map[string]string{"a": "1"}, true},
	}
	for _, test := range tests {
		t.Run(test.requestUrl, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			mockRequest := NewMock().Get(test.requestUrl).QueryParams(test.queryToMatch)
			matches := queryParamMatcher(req, mockRequest)
			assertEqual(t, test.shouldMatch, matches)
		})
	}
}

func TestMocks_QueryParams_DoesNotOverwriteQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.com/v2/path?b=2&a=1", nil)
	mockRequest := NewMock().
		Get("http://test.com").
		Query("b", "2").
		QueryParams(map[string]string{"a": "1"})

	matches := queryParamMatcher(req, mockRequest)

	assertEqual(t, 2, len(mockRequest.query))
	assertEqual(t, true, matches)
}

func TestMocks_SchemeMatcher(t *testing.T) {
	tests := []struct {
		requestUrl  string
		mockUrl     string
		shouldMatch bool
	}{
		{"http://test.com", "https://test.com", false},
		{"https://test.com", "https://test.com", true},
		{"https://test.com", "test.com", true},
		{"localhost:80", "localhost:80", true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.requestUrl, test.mockUrl), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			matches := schemeMatcher(req, NewMock().Get(test.mockUrl))
			if !matches == test.shouldMatch {
				t.Fatalf("mockUrl='%s' requestUrl='%s' shouldMatch=%v",
					test.mockUrl, test.requestUrl, test.shouldMatch)
			}
		})
	}
}

func TestMocks_BodyMatcher(t *testing.T) {
	tests := []struct {
		requestBody string
		matchBody   string
		shouldMatch bool
	}{
		{`{"a": 1}`, "", true},
		{``, `{"a": 1}`, false},
		{"golang\n", "go[lang]?", true},
		{"golang\n", "go[lang]?", true},
		{"golang", "goat", false},
		{"go\n", "go[lang]?", true},
		{`{"a":"12345"}\n`, `{"a":"12345"}`, true},
		{`{"a":"12345"}`, `{"b":"12345"}`, false},
		{`{"a":"12345"}`, `{"a":"12345"}`, true},
		{`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`,
			`{"b": [{"key": "c", "value": "result"}], "a": 12345}`, true},
	}

	for _, test := range tests {
		t.Run(test.matchBody, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(test.requestBody))
			matches := bodyMatcher(req, NewMock().Get("/path").Body(test.matchBody))
			assertEqual(t, test.shouldMatch, matches)
		})
	}
}

func TestMocks_PathMatcher(t *testing.T) {
	tests := []struct {
		requestUrl  string
		pathToMatch string
		shouldMatch bool
	}{
		{"http://test.com/v1/path", "/v1/path", true},
		{"http://test.com/v1/path", "/v1/not", false},
		{"http://test.com/v1/path", "", true},
		{"http://test.com", "", true},
		{"http://test.com/v2/path", "/v2/.+th", true},
	}
	for _, test := range tests {
		t.Run(test.pathToMatch, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestUrl, nil)
			matches := pathMatcher(req, NewMock().Get(test.pathToMatch))
			if !matches == test.shouldMatch {
				t.Fatalf("methodToMatch='%s' requestUrl='%s' shouldMatch=%v",
					test.pathToMatch, test.requestUrl, test.shouldMatch)
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

	mockResponse := matches(req, []*Mock{getUser, getPreferences})

	assertNotNil(t, mockResponse)
	assertEqual(t, `{"is_contactable": true}`, mockResponse.body)
}

func TestMocks_Matches_NilIfNoMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	mockResponse := matches(req, []*Mock{})

	if mockResponse != nil {
		t.Fatal("expected mockResponse to be nil")
	}
}

func TestMocks_MethodMatcher(t *testing.T) {
	tests := []struct {
		requestMethod string
		methodToMatch string
		shouldMatch   bool
	}{
		{http.MethodGet, http.MethodGet, true},
		{http.MethodDelete, "", true},
		{http.MethodPut, http.MethodGet, false},
	}
	for _, test := range tests {
		t.Run(test.requestMethod, func(t *testing.T) {
			req := httptest.NewRequest(test.requestMethod, "/path", nil)
			matches := methodMatcher(req, NewMock().Method(test.methodToMatch))
			if !matches == test.shouldMatch {
				t.Fatalf("methodToMatch='%s' requestMethod='%s' shouldMatch=%v",
					test.methodToMatch, test.requestMethod, test.shouldMatch)
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
			assertEqual(t, test.expectedMethod, mock.request.method)
		})
	}
}

func TestMocks_Response_Headers_WithNormalizedKeys(t *testing.T) {
	mockResponse := NewMock().
		Get("test").
		RespondWith().
		Header("a", "1").
		Headers(map[string]string{"B": "2"}).
		Header("c", "3")

	response := buildResponseFromMock(mockResponse)

	assertEqual(t, http.Header(map[string][]string{"A": {"1"}, "B": {"2"}, "C": {"3"}}), response.Header)
}

func TestMocks_Response_Cookies(t *testing.T) {
	mockResponse := NewMock().
		Get("test").
		RespondWith().
		Cookie("A", "1").
		Cookies(NewCookie("B").Value("2")).
		Cookie("C", "3")

	response := buildResponseFromMock(mockResponse)

	assertEqual(t, []*http.Cookie{
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
