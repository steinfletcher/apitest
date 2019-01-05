package apitest

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestApiTest_Assert_StatusCodes(t *testing.T) {
	tests := []struct {
		responseStatus []int
		assertFunc     Assert
	}{
		{[]int{200, 312, 399}, IsSuccess},
		{[]int{400, 404, 499}, IsClientError},
		{[]int{500, 503}, IsServerError},
	}
	for _, test := range tests {
		for _, status := range test.responseStatus {
			handler := http.NewServeMux()
			handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			})

			t.Run(fmt.Sprintf("status: %d", status), func(t *testing.T) {
				New().
					Handler(handler).
					Get("/hello").
					Expect(t).
					Assert(test.assertFunc).
					End()
			})
		}
	}
}

func TestApiTest_Assert_JSONPathContains(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(JSONPathContains(`$.b[? @.key=="c"].value`, "result")).
		End()
}

func TestApiTest_Assert_JSONPathIs_Numeric(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(JSONPathEqual(`$.a`, float64(12345))).
		End()
}

func TestApiTest_Assert_JSONPathEqual_String(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": "12345", "b": [{"key": "c", "value": "result"}]}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(JSONPathEqual(`$.a`, "12345")).
		End()
}

func TestApiTest_Assert_JSONPathEqual_Map(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": "hello", "b": 12345}`))
		if err != nil {
			panic(err)
		}
	})

	New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(JSONPathEqual(`$`, map[string]interface{}{"a": "hello", "b": float64(12345)})).
		End()
}

func Test_IncludesElement(t *testing.T) {
	list1 := []string{"Foo", "Bar"}
	list2 := []int{1, 2}
	simpleMap := map[interface{}]interface{}{"Foo": "Bar"}

	ok, found := includesElement("Hello World", "World")
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(list1, "Foo")
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(list1, "Bar")
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(list2, 1)
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(list2, 2)
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(list1, "Foo!")
	assert.True(t, ok)
	assert.False(t, found)

	ok, found = includesElement(list2, 3)
	assert.True(t, ok)
	assert.False(t, found)

	ok, found = includesElement(list2, "1")
	assert.True(t, ok)
	assert.False(t, found)

	ok, found = includesElement(simpleMap, "Foo")
	assert.True(t, ok)
	assert.True(t, found)

	ok, found = includesElement(simpleMap, "Bar")
	assert.True(t, ok)
	assert.False(t, found)

	ok, found = includesElement(1433, "1")
	assert.False(t, ok)
	assert.False(t, found)
}
