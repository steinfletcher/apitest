package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// IsSuccess is a convenience function to assert on a range of happy path status codes
var IsSuccess Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		return nil
	}
	return errors.New("not a client error. Status code=" + strconv.Itoa(response.StatusCode))
}

// IsClientError is a convenience function to assert on a range of client error status codes
var IsClientError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 400 && response.StatusCode < 500 {
		return nil
	}
	return errors.New("not a client error. Status code=" + strconv.Itoa(response.StatusCode))
}

// IsServerError is a convenience function to assert on a range of server error status codes
var IsServerError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 500 {
		return nil
	}
	return errors.New("not a server error. Status code=" + strconv.Itoa(response.StatusCode))
}

func assertEqual(t *testing.T, expected, actual interface{}, message ...string) {
	if !objectsAreEqual(expected, actual) {
		if len(message) > 0 {
			t.Fatalf(strings.Join(message, ","))
		} else {
			t.Fatalf("Expected %s but recevied %s", expected, actual)
		}
	}
}

func objectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func jsonEqual(t *testing.T, expected string, actual string) {
	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		t.Fatalf(fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()))
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		t.Fatalf(fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()))
	}

	assertEqual(t, expectedJSONAsInterface, actualJSONAsInterface)
}
