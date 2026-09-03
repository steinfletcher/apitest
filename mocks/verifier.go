package mocks

import (
	"github.com/steinfletcher/apitest"
)

var _ apitest.Verifier = MockVerifier{}

// MockVerifier is a mock of the Verifier interface that is used in tests of apitest
type MockVerifier struct {
	EqualFn      func(t apitest.TestingT, expected, actual any, msgAndArgs ...any) bool
	EqualInvoked bool

	TrueFn      func(t apitest.TestingT, val bool, msgAndArgs ...any) bool
	TrueInvoked bool

	JSONEqFn      func(t apitest.TestingT, expected string, actual string, msgAndArgs ...any) bool
	JSONEqInvoked bool

	FailFn      func(t apitest.TestingT, failureMessage string, msgAndArgs ...any) bool
	FailInvoked bool

	NoErrorFn      func(t apitest.TestingT, err error, msgAndArgs ...any) bool
	NoErrorInvoked bool
}

func NewVerifier() MockVerifier {
	return MockVerifier{
		EqualFn: func(t apitest.TestingT, expected, actual any, msgAndArgs ...any) bool {
			return true
		},
		JSONEqFn: func(t apitest.TestingT, expected string, actual string, msgAndArgs ...any) bool {
			return true
		},
		FailFn: func(t apitest.TestingT, failureMessage string, msgAndArgs ...any) bool {
			return true
		},
		NoErrorFn: func(t apitest.TestingT, err error, msgAndArgs ...any) bool {
			return true
		},
		TrueFn: func(t apitest.TestingT, val bool, msgAndArgs ...any) bool {
			return true
		},
	}
}

// Equal mocks the Equal method of the Verifier
func (m MockVerifier) Equal(t apitest.TestingT, expected, actual any, msgAndArgs ...any) bool {
	m.EqualInvoked = true
	return m.EqualFn(t, expected, actual, msgAndArgs)
}

// True mocks the Equal method of the Verifier
func (m MockVerifier) True(t apitest.TestingT, val bool, msgAndArgs ...any) bool {
	m.TrueInvoked = true
	return m.TrueFn(t, val, msgAndArgs)
}

// JSONEq mocks the JSONEq method of the Verifier
func (m MockVerifier) JSONEq(t apitest.TestingT, expected string, actual string, msgAndArgs ...any) bool {
	m.JSONEqInvoked = true
	return m.JSONEqFn(t, expected, actual, msgAndArgs)
}

// Fail mocks the Fail method of the Verifier
func (m MockVerifier) Fail(t apitest.TestingT, failureMessage string, msgAndArgs ...any) bool {
	m.FailInvoked = true
	return m.FailFn(t, failureMessage, msgAndArgs)
}

// NoError asserts that a function returned no error
func (m MockVerifier) NoError(t apitest.TestingT, err error, msgAndArgs ...any) bool {
	m.NoErrorInvoked = true
	return m.NoErrorFn(t, err, msgAndArgs)
}
