package apitest

import (
	"net/http"
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Drink string `json:"drink"`
}

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

func TestApiTest_Assert_AssertEquals_StringValue_WithMessage(t *testing.T) {
	assertEqual(t, "OneString", "OneString", "Should be equal")
}

func TestApiTest_Assert_AssertEquals_IntValue_WithoutMessage(t *testing.T) {
	assertEqual(t, 420, 420)
}

func TestApiTest_Assert_objectsAreEqual(t *testing.T) {
	if !objectsAreEqual(420, 420) {
		t.Fatalf("Objects should have been equal")
	}
}

func TestApiTest_Assert_objectsAreEqual_ExpectFalse(t *testing.T) {
	if objectsAreEqual(420, 421) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestApiTest_Assert_objectsAreEqual_MissmatchedType(t *testing.T) {
	if objectsAreEqual(420, testItem{"Tom", "Beer"}) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestApiTest_Assert_JsonEqual(t *testing.T) {
	jsonA := `{"name":"Tom","Drink":"Beer"}`
	jsonB := `{"name":"Tom","Drink":"Beer"}`

	jsonEqual(t, jsonA, jsonB)
}
