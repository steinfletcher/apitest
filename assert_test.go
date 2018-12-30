package apitest

import (
	"fmt"
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
				New(handler).
					Get("/hello").
					Expect(t).
					Assert(test.assertFunc).
					End()
			})
		}
	}
}
