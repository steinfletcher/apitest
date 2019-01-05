package apitest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
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

// JSONPathContains is a convenience function to assert that a jsonpath expression extracts a value in an array
func JSONPathContains(expression string, expected interface{}) Assert {
	return func(res *http.Response, req *http.Request) error {
		value, err := jsonPath(res.Body, expression)
		if err != nil {
			return err
		}

		ok, found := includesElement(value, expected)
		if !ok {
			return errors.New(fmt.Sprintf("\"%s\" could not be applied builtin len()", expected))
		}
		if !found {
			return errors.New(fmt.Sprintf("\"%s\" does not contain \"%s\"", expected, value))
		}
		return nil
	}
}

// JSONPathEqual is a convenience function to assert that a jsonpath expression extracts a value
func JSONPathEqual(expression string, expected interface{}) Assert {
	return func(res *http.Response, req *http.Request) error {
		value, err := jsonPath(res.Body, expression)
		if err != nil {
			return err
		}

		if !assert.ObjectsAreEqual(value, expected) {
			return errors.New(fmt.Sprintf("\"%s\" not equal to \"%s\"", value, expected))
		}
		return nil
	}
}

func jsonPath(reader io.Reader, expression string) (interface{}, error) {
	v := interface{}(nil)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}

	value, err := jsonpath.Get(expression, v)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// courtesy of github.com/stretchr/testify
func includesElement(list interface{}, element interface{}) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	defer func() {
		if e := recover(); e != nil {
			ok = false
			found = false
		}
	}()

	if reflect.TypeOf(list).Kind() == reflect.String {
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	if reflect.TypeOf(list).Kind() == reflect.Map {
		mapKeys := listValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if assert.ObjectsAreEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		if assert.ObjectsAreEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}
	return true, false
}
