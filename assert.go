package apitest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

// Verifier is the assertion interface allowing consumers to inject a custom assertion implementation.
// It also allows failure scenarios to be tested within apitest
type Verifier interface {
	Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool
	JSONEq(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool
	Fail(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool
	NoError(t *testing.T, err error, msgAndArgs ...interface{}) bool
}

type DefaultVerifier struct{}

// JSONEq asserts that two JSON strings are equivalent
func (a DefaultVerifier) JSONEq(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool {
	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		return fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()), msgAndArgs...)
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		return fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()), msgAndArgs...)
	}

	return a.Equal(t, expectedJSONAsInterface, actualJSONAsInterface, msgAndArgs...)
}

// Equal asserts that two objects are equal
func (a DefaultVerifier) Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if err := validateEqualArgs(expected, actual); err != nil {
		return fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			expected, actual, err), msgAndArgs...)
	}

	if !objectsAreEqual(expected, actual) {
		diff := diff(expected, actual)
		expected, actual = formatUnequalValues(expected, actual)
		return fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s%s", expected, actual, diff), msgAndArgs...)
	}

	return true
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

func formatUnequalValues(expected, actual interface{}) (e string, a string) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return fmt.Sprintf("%T(%s)", expected, truncatingFormat(expected)),
			fmt.Sprintf("%T(%s)", actual, truncatingFormat(actual))
	}
	switch expected.(type) {
	case time.Duration:
		return fmt.Sprintf("%v", expected), fmt.Sprintf("%v", actual)
	}
	return truncatingFormat(expected), truncatingFormat(actual)
}

func truncatingFormat(data interface{}) string {
	value := fmt.Sprintf("%#v", data)
	max := bufio.MaxScanTokenSize - 100 // Give us some space the type info too if needed.
	if len(value) > max {
		value = value[0:max] + "<... truncated>"
	}
	return value
}

func validateEqualArgs(expected, actual interface{}) error {
	if expected == nil && actual == nil {
		return nil
	}

	if isFunction(expected) || isFunction(actual) {
		return errors.New("cannot take func type as argument")
	}
	return nil
}

func isFunction(arg interface{}) bool {
	if arg == nil {
		return false
	}
	return reflect.TypeOf(arg).Kind() == reflect.Func
}

// Fail reports a failure
func (a DefaultVerifier) Fail(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool {
	content := []labeledContent{
		{"Error Trace", strings.Join(callerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
	}

	// TODO add this in after implementing TestingT
	//// Add test name if the Go version supports it
	//if n, ok := t.(interface {
	//	Name() string
	//}); ok {
	//	content = append(content, labeledContent{"Test", n.Name()})
	//}

	message := messageFromMsgAndArgs(msgAndArgs...)
	if len(message) > 0 {
		content = append(content, labeledContent{"Messages", message})
	}

	t.Errorf("\n%s", ""+labeledOutput(content...))

	return false
}

// NoError asserts that a function returned no error
func (a DefaultVerifier) NoError(t *testing.T, err error, msgAndArgs ...interface{}) bool {
	if err != nil {
		return fail(t, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}
	return true
}

func fail(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool {
	content := []labeledContent{
		{"Error Trace", strings.Join(callerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
	}

	// TODO add this in after implementing TestingT
	//// Add test name if the Go version supports it
	//if n, ok := t.(interface {
	//	Name() string
	//}); ok {
	//	content = append(content, labeledContent{"Test", n.Name()})
	//}

	message := messageFromMsgAndArgs(msgAndArgs...)
	if len(message) > 0 {
		content = append(content, labeledContent{"Messages", message})
	}

	t.Errorf("\n%s", ""+labeledOutput(content...))

	return false
}

func labeledOutput(content ...labeledContent) string {
	longestLabel := 0
	for _, v := range content {
		if len(v.label) > longestLabel {
			longestLabel = len(v.label)
		}
	}
	var output string
	for _, v := range content {
		output += "\t" + v.label + ":" + strings.Repeat(" ", longestLabel-len(v.label)) + "\t" + indentMessageLines(v.content, longestLabel) + "\n"
	}
	return output
}

// Stolen from `testify`
func callerInfo() []string {
	var pc uintptr
	var ok bool
	var file string
	var line int
	var name string

	callers := []string{}
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			// The breaks below failed to terminate the loop, and we ran off the
			// end of the call stack.
			break
		}

		// This is a huge edge case, but it will panic if this is the case, see #180
		if file == "<autogenerated>" {
			break
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		// testing.tRunner is the standard library function that calls
		// tests. Subtests are called directly by tRunner, without going through
		// the Test/Benchmark/Example function that contains the t.Run calls, so
		// with subtests we should break when we hit tRunner, without adding it
		// to the list of callers.
		if name == "testing.tRunner" {
			break
		}

		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		if len(parts) > 1 {
			dir := parts[len(parts)-2]
			if (dir != "assert" && dir != "mock" && dir != "require") || file == "mock_test.go" {
				callers = append(callers, fmt.Sprintf("%s:%d", file, line))
			}
		}

		// Drop the package
		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}

	return callers
}

func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}

func indentMessageLines(message string, longestLabelLen int) string {
	outBuf := new(bytes.Buffer)

	for i, scanner := 0, bufio.NewScanner(strings.NewReader(message)); scanner.Scan(); i++ {
		// no need to align first line because it starts at the correct location (after the label)
		if i != 0 {
			// append alignLen+1 spaces to align with "{{longestLabel}}:" before adding tab
			outBuf.WriteString("\n\t" + strings.Repeat(" ", longestLabelLen+1) + "\t")
		}
		outBuf.WriteString(scanner.Text())
	}

	return outBuf.String()
}

type labeledContent struct {
	label   string
	content string
}

func newDefaultVerifier() Verifier {
	return DefaultVerifier{}
}

// NoopVerifier is a verifier that does not perform verification
type NoopVerifier struct{}

var _ Verifier = NoopVerifier{}

// Equal does not perform any assertion and always returns true
func (n NoopVerifier) Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	return true
}

// JSONEq does not perform any assertion and always returns true
func (n NoopVerifier) JSONEq(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool {
	return true
}

// Fail does not perform any assertion and always returns true
func (n NoopVerifier) Fail(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool {
	return true
}

// NoError asserts that a function returned no error
func (n NoopVerifier) NoError(t *testing.T, err error, msgAndArgs ...interface{}) bool {
	return true
}

// IsSuccess is a convenience function to assert on a range of happy path status codes
var IsSuccess Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("not success. Status code=%d", response.StatusCode)
}

// IsClientError is a convenience function to assert on a range of client error status codes
var IsClientError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 400 && response.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("not a client error. Status code=%d", response.StatusCode)
}

// IsServerError is a convenience function to assert on a range of server error status codes
var IsServerError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= 500 {
		return nil
	}
	return fmt.Errorf("not a server error. Status code=%d", response.StatusCode)
}
