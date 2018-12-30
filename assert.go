package apitest

import (
	"errors"
	"net/http"
	"strconv"
)

var IsSuccess Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		return nil
	}
	return errors.New("not a client error. Status code=" + strconv.Itoa(response.StatusCode))
}

var IsClientError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 400 && response.StatusCode < 500 {
		return nil
	}
	return errors.New("not a client error. Status code=" + strconv.Itoa(response.StatusCode))
}

var IsServerError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 500 {
		return nil
	}
	return errors.New("not a server error. Status code=" + strconv.Itoa(response.StatusCode))
}
