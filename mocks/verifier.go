package mocks

import (
	"github.com/steinfletcher/apitest"
	"testing"
)

var _ apitest.Verifier = MockVerifier{}

// MockVerifier is a mock of the Verifier interface that is used in tests of apitest
type MockVerifier struct {
	EqualFn      func(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool
	EqualInvoked bool

	JSONEqFn      func(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool
	JSONEqInvoked bool

	FailFn      func(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool
	FailInvoked bool

	NoErrorFn      func(t *testing.T, err error, msgAndArgs ...interface{}) bool
	NoErrorInvoked bool
}

func NewVerifier() MockVerifier {
	return MockVerifier{
		EqualFn: func(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			return true
		},
		JSONEqFn: func(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool {
			return true
		},
		FailFn: func(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool {
			return true
		},
		NoErrorFn: func(t *testing.T, err error, msgAndArgs ...interface{}) bool {
			return true
		},
	}
}

// Equal mocks the Equal method of the Verifier
func (m MockVerifier) Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	m.EqualInvoked = true
	return m.EqualFn(t, expected, actual, msgAndArgs)
}

// JSONEq mocks the JSONEq method of the Verifier
func (m MockVerifier) JSONEq(t *testing.T, expected string, actual string, msgAndArgs ...interface{}) bool {
	m.JSONEqInvoked = true
	return m.JSONEqFn(t, expected, actual, msgAndArgs)
}

// Fail mocks the Fail method of the Verifier
func (m MockVerifier) Fail(t *testing.T, failureMessage string, msgAndArgs ...interface{}) bool {
	m.FailInvoked = true
	return m.FailFn(t, failureMessage, msgAndArgs)
}

// NoError asserts that a function returned no error
func (m MockVerifier) NoError(t *testing.T, err error, msgAndArgs ...interface{}) bool {
	m.NoErrorInvoked = true
	return m.NoErrorFn(t, err, msgAndArgs)
}
