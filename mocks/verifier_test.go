package mocks_test

import (
	"errors"
	"testing"

	"github.com/steinfletcher/apitest"
	"github.com/steinfletcher/apitest/mocks"
)

func TestNewVerifier_DefaultsPassEveryAssertion(t *testing.T) {
	verifier := mocks.NewVerifier()

	if !verifier.Equal(t, 1, 2) {
		t.Fatal("expected default Equal to pass")
	}
	if !verifier.True(t, false) {
		t.Fatal("expected default True to pass")
	}
	if !verifier.JSONEq(t, `{"a": 1}`, `{"b": 2}`) {
		t.Fatal("expected default JSONEq to pass")
	}
	if !verifier.Fail(t, "boom") {
		t.Fatal("expected default Fail to pass")
	}
	if !verifier.NoError(t, errors.New("boom")) {
		t.Fatal("expected default NoError to pass")
	}
}

func TestMockVerifier_DelegatesToTheConfiguredFunctions(t *testing.T) {
	verifier := mocks.NewVerifier()

	var equalArgs, trueArgs, jsonEqArgs, failArgs, noErrorArgs []any
	verifier.EqualFn = func(t apitest.TestingT, expected, actual any, msgAndArgs ...any) bool {
		equalArgs = []any{expected, actual}
		return false
	}
	verifier.TrueFn = func(t apitest.TestingT, val bool, msgAndArgs ...any) bool {
		trueArgs = []any{val}
		return false
	}
	verifier.JSONEqFn = func(t apitest.TestingT, expected, actual string, msgAndArgs ...any) bool {
		jsonEqArgs = []any{expected, actual}
		return false
	}
	verifier.FailFn = func(t apitest.TestingT, failureMessage string, msgAndArgs ...any) bool {
		failArgs = []any{failureMessage}
		return false
	}
	verifier.NoErrorFn = func(t apitest.TestingT, err error, msgAndArgs ...any) bool {
		noErrorArgs = []any{err}
		return false
	}

	err := errors.New("boom")
	results := []bool{
		verifier.Equal(t, "expected", "actual"),
		verifier.True(t, true),
		verifier.JSONEq(t, `{"a": 1}`, `{"a": 2}`),
		verifier.Fail(t, "failure message"),
		verifier.NoError(t, err),
	}

	for i, result := range results {
		if result {
			t.Fatalf("expected configured function %d to control the result", i)
		}
	}
	assertDeepEqual(t, []any{"expected", "actual"}, equalArgs)
	assertDeepEqual(t, []any{true}, trueArgs)
	assertDeepEqual(t, []any{`{"a": 1}`, `{"a": 2}`}, jsonEqArgs)
	assertDeepEqual(t, []any{"failure message"}, failArgs)
	assertDeepEqual(t, []any{err}, noErrorArgs)
}

// The mock forwards msgAndArgs as a single slice argument rather than spreading it,
// so a configured function sees it as msgAndArgs[0].
func TestMockVerifier_ForwardsMessageArgsAsASingleSlice(t *testing.T) {
	verifier := mocks.NewVerifier()

	var received []any
	verifier.EqualFn = func(t apitest.TestingT, expected, actual any, msgAndArgs ...any) bool {
		received = msgAndArgs
		return true
	}

	verifier.Equal(t, 1, 1, "message", 42)

	if len(received) != 1 {
		t.Fatalf("expected a single forwarded argument, got %d", len(received))
	}
	assertDeepEqual(t, []any{"message", 42}, received[0])
}

func assertDeepEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !apitest.DefaultVerifier.Equal(apitest.DefaultVerifier{}, t, expected, actual) {
		t.FailNow()
	}
}
