package apitest

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestApiTest_AddsJSONBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)
		if string(bytes) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsTextBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)
		if string(bytes) != `hello` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
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

	New(handler).
		Get("/hello").
		Query(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsHeadersToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "12345" != r.Header.Get("My-Header") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Delete("/hello").
		Headers(map[string]string{"My-Header": "12345"}).
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
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Method(http.MethodGet).
		URL("/hello").
		Cookies(map[string]string{"Cookie1": "Yummy"}).
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

	New(handler).
		Get("/hello").
		BasicAuth("username:password").
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

	New(handler).
		Get("/hello").
		Expect(t).
		Body(`{"a": 12345}`).
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

	New(handler).
		Get("/hello").
		Expect(t).
		Body(`hello`).
		Status(http.StatusOK).
		End()
}

func TestApiTest_MatchesResponseCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(map[string]string{
			"ABC": "12345",
			"DEF": "67890",
		}).
		CookiePresent("XXX").
		CookiePresent("VVV").
		End()
}

func TestApiTest_MatchesResponseHttpCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "ABC=12345; DEF=67890;")
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Get("/hello").
		Expect(t).
		HttpCookies([]http.Cookie{
			{Name: "ABC", Value: "12345"},
			{Name: "DEF", Value: "67890"},
		}).
		End()
}

func TestApiTest_MatchesResponseHeaders(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ABC", "12345")
		w.Header().Set("DEF", "67890")
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{
			"ABC": "12345",
			"DEF": "67890",
		}).
		End()
}

func TestApiTest_SupportsJSONPathExpectations(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`))
		if err != nil {
			panic(err)
		}
	})

	New(handler).
		Get("/hello").
		Expect(t).
		JSONPath(`$.b[? @.key=="c"].value`, func(values interface{}) {
			assert.Contains(t, values, "result")
		}).
		End()
}

func TestApiTest_Observe(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	New(handler).
		Observe(func(res *http.Response, req *http.Request) {
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/hello", req.URL.Path)
		}).
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

	New(handler).
		Observe(DumpHttp).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusCreated).
		End()
}
