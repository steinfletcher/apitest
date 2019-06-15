package apitest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApiTest_AddsJSONBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		if string(data) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Header("Content-Type", "application/json").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsJSONBodyToRequestUsingJSON(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		if string(data) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Post("/hello").
		JSON(`{"a": 12345}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsTextBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		if string(data) != `hello` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Put("/hello").
		Body(`hello`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamsToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "b" != r.URL.Query().Get("a") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		QueryParams(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamCollectionToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "a=b&a=c&a=d&e=f" != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{"a": {"b", "c", "d"}}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamCollectionToRequest_HandlesEmpty(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "e=f" != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_CanCombineQueryParamMethods(t *testing.T) {
	expectedQueryString := "a=1&a=2&a=9&a=22&b=2"
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if expectedQueryString != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		Query("a", "9").
		Query("a", "22").
		QueryCollection(map[string][]string{"a": {"1", "2"}}).
		QueryParams(map[string]string{"b": "2"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsHeadersToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		header := r.Header["Authorization"]
		if len(header) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Delete("/hello").
		Headers(map[string]string{"Authorization": "12345"}).
		Header("Authorization", "098765").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsContentTypeHeaderToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Post("/hello").
		ContentType("application/x-www-form-urlencoded").
		Body(`name=John`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsCookiesToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("Cookie1"); err != nil || cookie.Value != "Yummy" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if cookie, err := r.Cookie("Cookie"); err != nil || cookie.Value != "Nom" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Method(http.MethodGet).
		URL("/hello").
		Cookie("Cookie", "Nom").
		Cookies(NewCookie("Cookie1").Value("Yummy")).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsBasicAuthToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if username != "username" || password != "password" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New("some test name").
		Handler(handler).
		Get("/hello").
		BasicAuth("username", "password").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_MatchesJSONResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{"a": 12345}`).
		Status(http.StatusCreated).
		End()
}

func TestApiTest_MatchesJSONResponseBodyWithWhitespace(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{
			"a": 12345,
			"b": "hi"
		}`).
		Status(http.StatusCreated).
		End()
}

func TestApiTest_MatchesTextResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte(`hello`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`hello`).
		Status(http.StatusOK).
		End()
}

func TestApiTest_MatchesResponseCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-ExpectedCookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		http.SetCookie(w, &http.Cookie{
			Name:  "ABC",
			Value: "12345",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "DEF",
			Value: "67890",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "XXX",
			Value: "1fsadg235",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "VVV",
			Value: "9ig32g34g",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "YYY",
			Value: "kfiufhtne",
		})

		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(
			NewCookie("ABC").Value("12345"),
			NewCookie("DEF").Value("67890")).
		Cookie("YYY", "kfiufhtne").
		CookiePresent("XXX").
		CookiePresent("VVV").
		CookieNotPresent("ZZZ").
		CookieNotPresent("TomBeer").
		End()
}

func TestApiTest_MatchesResponseHttpCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "ABC",
			Value: "12345",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "DEF",
			Value: "67890",
		})
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			NewCookie("ABC").Value("12345"),
			NewCookie("DEF").Value("67890")).
		End()
}

func TestApiTest_MatchesResponseHttpCookies_OnlySuppliedFields(t *testing.T) {
	parsedDateTime, err := time.Parse(time.RFC3339, "2019-01-26T23:19:02Z")
	if err != nil {
		t.Fatalf("%s", err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "pdsanjdna_8e8922",
			Path:     "/",
			Expires:  parsedDateTime,
			Secure:   true,
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			NewCookie("session_id").
				Value("pdsanjdna_8e8922").
				Path("/").
				Expires(parsedDateTime).
				Secure(true).
				HttpOnly(true)).
		End()
}

func TestApiTest_MatchesResponseHeaders_WithMixedKeyCase(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ABC", "12345")
		w.Header().Set("DEF", "67890")
		w.Header().Set("Authorization", "12345")
		w.Header().Add("authorizATION", "00000")
		w.Header().Add("Authorization", "98765")
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{
			"Abc": "12345",
			"Def": "67890",
		}).
		Header("Authorization", "12345").
		Header("Authorization", "00000").
		Header("authorization", "98765").
		HeaderPresent("Def").
		HeaderPresent("Authorization").
		HeaderNotPresent("XYZ").
		End()
}

func TestApiTest_EndReturnsTheResult(t *testing.T) {
	type resBody struct {
		B string `json:"b"`
	}
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`))
		if err != nil {
			panic(err)
		}
	})

	var r resBody
	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{
			"a": 12345,
			"b": "hi"
		}`).
		Status(http.StatusCreated).
		End().
		JSON(&r)

	assert.Equal(t, "hi", r.B)
}

func TestApiTest_CustomAssert(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-ExpectedCookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Assert(IsSuccess).
		End()
}

func TestApiTest_SupportsMultipleCustomAsserts(t *testing.T) {
	test := New().
		Patch("/hello").
		Expect(t).
		Assert(IsSuccess).
		Assert(IsSuccess)

	assert.Len(t, test.assert, 2)
}

func TestApiTest_AssertFunc(t *testing.T) {
	tests := []struct {
		statusCode  int
		expectedErr error
	}{
		{200, nil},
		{400, errors.New("not success. Status code=400")},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("status: %d", test.statusCode), func(t *testing.T) {
			res := httptest.NewRecorder()
			res.Code = test.statusCode
			apitTest := New().
				Patch("/hello").
				Expect(t).
				Assert(IsSuccess)

			err := apitTest.apiTest.assertFunc(res.Result(), httptest.NewRequest("GET", "/", nil))

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestApiTest_Report(t *testing.T) {
	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	reporter := &RecorderCaptor{}

	New("some test").
		Debug().
		Meta(map[string]interface{}{"host": "abc.com"}).
		Report(reporter).
		Mocks(getUser).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusOK).
		End()

	r := reporter.capturedRecorder
	assert.Equal(t, "POST /hello", r.Title)
	assert.Equal(t, "some test", r.SubTitle)
	assert.Len(t, r.Events, 4)
	assert.Equal(t, 200, r.Meta["status_code"])
	assert.Equal(t, "/hello", r.Meta["path"])
	assert.Equal(t, "POST", r.Meta["method"])
	assert.Equal(t, "some test", r.Meta["name"])
	assert.Equal(t, "abc.com", r.Meta["host"])
}

func TestApiTest_Recorder(t *testing.T) {
	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	reporter := &RecorderCaptor{}
	messageRequest := MessageRequest{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	messageResponse := MessageResponse{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	recorder := NewTestRecorder()
	recorder.AddMessageRequest(messageRequest)
	recorder.AddMessageResponse(messageResponse)

	New("some test").
		Meta(map[string]interface{}{"host": "abc.com"}).
		Report(reporter).
		Recorder(recorder).
		Mocks(getUser).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusOK).
		End()

	r := reporter.capturedRecorder
	assert.Len(t, r.Events, 6)
	assert.Equal(t, messageRequest, r.Events[0])
	assert.Equal(t, messageResponse, r.Events[1])
}

func TestApiTest_Observe(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	New("observe test").
		Observe(func(res *http.Response, req *http.Request, apiTest *APITest) {
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/hello", req.URL.Path)
			assert.Equal(t, "observe test", apiTest.name)
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_Observe_DumpsTheHttpRequestAndResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusCreated).
		End()
}

func TestApiTest_Intercept(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "a[]=xxx&a[]=yyy" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Header.Get("Auth-Token") != "12345" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Intercept(func(req *http.Request) {
			req.URL.RawQuery = "a[]=xxx&a[]=yyy"
			req.Header.Set("Auth-Token", req.Header.Get("authtoken"))
		}).
		Get("/hello").
		Headers(map[string]string{"authtoken": "12345"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_ReplicatesMocks(t *testing.T) {
	tests := []struct {
		times int
	}{
		{1}, {15}, {0}, {2},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.times), func(t *testing.T) {
			mock := NewMock().
				Get("/abc").
				RespondWith().
				Status(http.StatusOK).
				Times(test.times).
				End()

			numMocks := len(New().Mocks(mock).mocks)
			if numMocks != test.times {
				t.Fatalf("expected %d instances of the mock to be defined, but was %d", test.times, numMocks)
			}
		})
	}
}

func TestApiTest_ExposesRequestAndResponse(t *testing.T) {
	apiTest := New()

	assert.NotNil(t, apiTest.Request())
	assert.NotNil(t, apiTest.Response())
}

func TestApiTest_BuildQueryCollection(t *testing.T) {
	queryParams := map[string][]string{
		"a": {"22", "33"},
		"b": {"11"},
		"c": {},
	}

	params := buildQueryCollection(queryParams)

	expectedPairs := []pair{
		{l: "a", r: "22"},
		{l: "a", r: "33"},
		{l: "b", r: "11"},
	}

	if len(expectedPairs) != len(params) {
		t.Fatalf("Expected lengths not the same")
	}

	// Filter out expected pairs when found and remove
	for _, param := range params {
		for i, expectedPair := range expectedPairs {
			if reflect.DeepEqual(param, expectedPair) {
				expectedPairs = append(expectedPairs[:i], expectedPairs[i+1:]...)
			}
		}
	}

	if len(expectedPairs) != 0 {
		t.Fatalf("%s not found in params", expectedPairs)
	}
}

func TestApiTest_BuildQueryCollection_EmptyIfNoParams(t *testing.T) {
	queryParams := map[string][]string{"c": {}}

	params := buildQueryCollection(queryParams)

	if len(params) > 0 {
		t.Fatalf("Expected params to be empty")
	}
}

func TestApiTest_CopyHttpRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", bytes.NewBufferString("12345"))
	req.Header.Add("a", "1")
	req.Header.Add("b", "2")

	reqCopy := copyHttpRequest(req)

	assert.Equal(t, reqCopy.Method, req.Method)
	assert.Equal(t, reqCopy.URL, req.URL)
	assert.Equal(t, reqCopy.Host, req.Host)
	assert.Equal(t, reqCopy.ContentLength, req.ContentLength)
	assert.Equal(t, reqCopy.Header, req.Header)
	assert.Equal(t, reqCopy.Body, req.Body)
}

func TestCreateHash_GroupsByEndpoint(t *testing.T) {
	tests := []struct {
		app      string
		method   string
		path     string
		name     string
		expected string
	}{
		{app: "a", method: "GET", path: "/v1/abc", name: "test1", expected: "1850189403_2569220284"},
		{app: "a", method: "GET", path: "/v1/abc", name: "test2", expected: "1850189403_2619553141"},
		{app: "b", method: "GET", path: "/v1/abc", name: "test1", expected: "547235502_2569220284"},
		{app: "b", method: "GET", path: "/v1/abc", name: "test2", expected: "547235502_2619553141"},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s %s %s", test.app, test.method, test.path, test.name), func(t *testing.T) {
			meta := map[string]interface{}{
				"app":    test.app,
				"method": test.method,
				"path":   test.path,
				"name":   test.name,
			}
			hash := createHash(meta)
			assert.Equal(t, test.expected, hash)
		})
	}
}

// TestRealNetworking creates a server with two endpoints, /login sets a token via a cookie and /authenticated_resource
// validates the token. A cookie jar is used to verify session persistence across multiple apitest instances
func TestRealNetworking(t *testing.T) {
	srv := &http.Server{Addr: ":9876"}
	finish := make(chan struct{})
	tokenValue := "ABCDEF"
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "Token", Value: tokenValue})
		w.WriteHeader(203)
	})
	http.HandleFunc("/authenticated_resource", func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("Token")
		if err == http.ErrNoCookie {
			w.WriteHeader(400)
			return
		}
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if token.Value != tokenValue {
			t.Fatalf("token did not equal %s", tokenValue)
		}
		w.WriteHeader(204)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
		}()

		cookieJar, _ := cookiejar.New(nil)
		cli := &http.Client{
			Timeout: time.Second * 1,
			Jar:     cookieJar,
		}

		New().
			EnableNetworking(cli).
			Get("http://localhost:9876/login").
			Expect(t).
			Status(203).
			End()

		New().
			EnableNetworking(cli).
			Get("http://localhost:9876/authenticated_resource").
			Expect(t).
			Status(204).
			End()

		finish <- struct{}{}
	}()
	<-finish
}

func TestApiTest_AddsUrlEncodedFormBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		expectedPostFormData := map[string][]string{
			"name":     {"John"},
			"age":      {"99"},
			"children": {"Jack", "Ann"},
			"pets":     {"Toby", "Henry", "Alice"},
		}

		for key := range expectedPostFormData {
			if !reflect.DeepEqual(expectedPostFormData[key], r.PostForm[key]) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Post("/hello").
		FormData("name", "John").
		FormData("age", "99").
		FormData("children", "Jack").
		FormData("children", "Ann").
		FormData("pets", "Toby", "Henry", "Alice").
		Expect(t).
		Status(http.StatusOK).
		End()
}

type RecorderCaptor struct {
	capturedRecorder Recorder
}

func (r *RecorderCaptor) Format(recorder *Recorder) {
	r.capturedRecorder = *recorder
}
