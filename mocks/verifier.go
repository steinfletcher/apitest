package mocks

import (
	"github.com/steinfletcher/apitest"
)

var _ apitest.Verifier = MockVerifier{}

// MockVerifier is a mock of the Verifier interface that is used in tests of apitest
type MockVerifier struct {
	EqualFn      func(t apitest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool
	EqualInvoked bool

	TrueFn      func(t apitest.TestingT, val bool, msgAndArgs ...interface{}) bool
	TrueInvoked bool

	JSONEqFn      func(t apitest.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool
	JSONEqInvoked bool

	FailFn      func(t apitest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool
	FailInvoked bool

	NoErrorFn      func(t apitest.TestingT, err error, msgAndArgs ...interface{}) bool
	NoErrorInvoked bool
}

func NewVerifier() MockVerifier {
	return MockVerifier{
		EqualFn: func(t apitest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			return true
		},
		JSONEqFn: func(t apitest.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool {
			return true
		},
		FailFn: func(t apitest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
			return true
		},
		NoErrorFn: func(t apitest.TestingT, err error, msgAndArgs ...interface{}) bool {
			return true
		},
		TrueFn: func(t apitest.TestingT, val bool, msgAndArgs ...interface{}) bool {
			return true
		},
	}
}

// Equal mocks the Equal method of the Verifier
func (m MockVerifier) Equal(t apitest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	m.EqualInvoked = true
	return m.EqualFn(t, expected, actual, msgAndArgs)
}

// True mocks the Equal method of the Verifier
func (m MockVerifier) True(t apitest.TestingT, val bool, msgAndArgs ...interface{}) bool {
	m.TrueInvoked = true
	return m.TrueFn(t, val, msgAndArgs)
}

// JSONEq mocks the JSONEq method of the Verifier
func (m MockVerifier) JSONEq(t apitest.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool {
	m.JSONEqInvoked = true
	return m.JSONEqFn(t, expected, actual, msgAndArgs)
}

// Fail mocks the Fail method of the Verifier
func (m MockVerifier) Fail(t apitest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
	m.FailInvoked = true
	return m.FailFn(t, failureMessage, msgAndArgs)
}

// NoError asserts that a function returned no error
func (m MockVerifier) NoError(t apitest.TestingT, err error, msgAndArgs ...interface{}) bool {
	m.NoErrorInvoked = true
	return m.NoErrorFn(t, err, msgAndArgs)
}
