package assert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func Len(t testing.TB, actual interface{}, expected int) {
	v := reflect.ValueOf(actual)
	defer func() {
		if e := recover(); e != nil {
			t.Fatal("could not determine length of actual")
		}
	}()

	if v.Len() != expected {
		t.Fatalf("expected len to be %d but was %d", expected, 1)
	}
}

func Error(t testing.TB, actual error, expected string) {
	if actual == nil {
		t.Fatalf("expected an error but got nil.")
	}

	if expected != actual.Error() {
		t.Fatalf("Error message not equal:\nexpected: %q\nactual  : %q", expected, actual)
	}
}

func True(t testing.TB, actual bool) {
	if !actual {
		t.Fatal("expected true but received false")
	}
}

func False(t testing.TB, actual bool) {
	if actual {
		t.Fatal("expected false but received true")
	}
}

func Equal(t testing.TB, expected, actual interface{}, message ...string) {
	if !ObjectsAreEqual(expected, actual) {
		if len(message) > 0 {
			t.Fatalf(strings.Join(message, ", "))
		} else {
			t.Fatalf("Expected '%+v' but recevied '%+v'", expected, actual)
		}
	}
}

func ObjectsAreEqual(expected, actual interface{}) bool {
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

func JsonEqual(t testing.TB, expected string, actual string) {
	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		t.Fatalf(fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()))
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		t.Fatalf(fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()))
	}

	Equal(t, expectedJSONAsInterface, actualJSONAsInterface)
}

func NotNil(t testing.TB, actual interface{}) {
	if actual == nil {
		t.Fatalf("Expected value to not be nil")
	}
}

func Nil(t testing.TB, actual interface{}) {
	if actual != nil {
		t.Fatalf("Expected value to be nil")
	}
}
