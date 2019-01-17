package apitest

import (
	"net/http"
	"testing"
)

func TestApiTest_Assert_StatusCodes(t *testing.T) {
	tests := []struct {
		responseStatus []int
		assertFunc     Assert
		isSuccess      bool
	}{
		{[]int{200, 312, 399}, IsSuccess, true},
		{[]int{400, 404, 499}, IsClientError, true},
		{[]int{500, 503}, IsServerError, true},
		{[]int{400, 500}, IsSuccess, false},
		{[]int{200, 500}, IsClientError, false},
		{[]int{200, 400}, IsServerError, false},
	}
	for _, test := range tests {
		for _, status := range test.responseStatus {
			response := &http.Response{StatusCode: status}
			err := test.assertFunc(response, nil)
			if test.isSuccess && err != nil {
				t.Fatalf("Expecteted nil but received %s", err)
			} else if !test.isSuccess && err == nil {
				t.Fatalf("Expected error but didn't receive one")
			}
		}
	}
}
