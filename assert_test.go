package apitest

import (
	"github.com/stretchr/testify/assert"
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
			if test.isSuccess {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		}
	}
}
