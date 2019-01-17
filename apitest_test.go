package apitest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"
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

	New().
		Handler(handler).
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
		Query(map[string]string{"a": "b"}).
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
		Query(map[string]string{"e": "f"}).
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
		Query(map[string]string{"e": "f"}).
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

func TestApiTest_AddsCookiesToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("Cookie1"); err != nil || cookie.Value != "Yummy" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Method(http.MethodGet).
		URL("/hello").
		Cookies(NewCookie("Cookie1").
			Value("Yummy")).
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

	New().
		Handler(handler).
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

func TestApiTest_MatchesResponseHeaders(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ABC", "12345")
		w.Header().Set("DEF", "67890")
		w.WriteHeader(http.StatusOK)
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{
			"ABC": "12345",
			"DEF": "67890",
		}).
		End()
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

func TestApiTest_Observe(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	New().
		Observe(func(res *http.Response, req *http.Request) {
			assertEqual(t, http.StatusOK, res.StatusCode)
			assertEqual(t, "/hello", req.URL.Path)
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
		Get("/hello").
		Intercept(func(req *http.Request) {
			req.URL.RawQuery = "a[]=xxx&a[]=yyy"
			req.Header.Set("Auth-Token", req.Header.Get("authtoken"))
		}).
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

	assertNotNil(t, apiTest.Request())
	assertNotNil(t, apiTest.Response())
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

	//Filter out expected pairs when found and remove
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

func assertNotNil(t *testing.T, actual interface{}) {
	if actual == nil {
		t.Fatalf("Expected value to not be nil")
	}
}
