package apitest

import (
	"errors"
	"net/http"
	"strconv"
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
