package assert

import (
	"errors"
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Drink string `json:"drink"`
}

type mockT struct {
	*testing.T
	failed bool
}

func (r *mockT) Fatalf(format string, args ...interface{}) {
	r.failed = true
}

func (r *mockT) Fatal(args ...interface{}) {
	r.failed = true
}

func TestAssert_AssertEquals_StringValue_WithMessage(t *testing.T) {
	Equal(t, "OneString", "OneString", "Should be equal")
}

func TestAssert_AssertEquals_IntValue_WithoutMessage(t *testing.T) {
	Equal(t, 420, 420)
}

func TestAssert_NotEqual(t *testing.T) {
	m := &mockT{}

	Equal(m, 420, 411)

	True(t, m.failed)
}

func TestAssert_NilFails(t *testing.T) {
	m := &mockT{}

	Nil(m, "abc")

	True(t, m.failed)
}

func TestAssert_NotNilFails(t *testing.T) {
	m := &mockT{}

	NotNil(m, nil)

	True(t, m.failed)
}

func TestAssert_ErrorFails(t *testing.T) {
	m := &mockT{}

	Error(m, errors.New("some error"), "err")

	True(t, m.failed)
}

func TestAssert_ObjectsAreEqual(t *testing.T) {
	if !ObjectsAreEqual(420, 420) {
		t.Fatalf("Objects should have been equal")
	}
}

func TestAssert_ObjectsAreEqual_ExpectFalse(t *testing.T) {
	if ObjectsAreEqual(420, 421) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestAssert_ObjectsAreEqual_MissmatchedType(t *testing.T) {
	if ObjectsAreEqual(420, testItem{"Tom", "Beer"}) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestAssert_ObjectsAreEqual_BytesEqual(t *testing.T) {
	if !ObjectsAreEqual([]byte("i_am_worraz"), []byte("i_am_worraz")) {
		t.Fatalf("Objects should have been equal")
	}
}

func TestAssert_ObjectsAreEqual_BytesStringNotEqual(t *testing.T) {
	if ObjectsAreEqual([]byte("i_am_worraz"), "i_am_worraz") {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestAssert_ObjectsAreEqual_BytesNotEqual(t *testing.T) {
	if ObjectsAreEqual([]byte("i_am_worraz"), []byte("the_emu")) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestAssert_ObjectsAreEqual_TrueIfBothNil(t *testing.T) {
	if !ObjectsAreEqual(nil, nil) {
		t.Fatalf("Objects should have been equal")
	}
}

func TestAssert_JsonEqual(t *testing.T) {
	jsonA := `{"name":"Tom","Drink":"Beer"}`
	jsonB := `{"name":"Tom","Drink":"Beer"}`

	JsonEqual(t, jsonA, jsonB)
}

func TestAssert_True(t *testing.T) {
	True(t, true)
}

func TestAssert_TrueFails(t *testing.T) {
	m := &mockT{}

	True(m, false)

	True(t, m.failed)
}

func TestAssert_False(t *testing.T) {
	False(t, false)
}

func TestAssert_FalseFails(t *testing.T) {
	m := &mockT{}

	False(m, true)

	True(t, m.failed)
}

func TestAssert_Len(t *testing.T) {
	Len(t, []string{}, 0)
	Len(t, []string{"1"}, 1)
	Len(t, []string{"1", "4", "51"}, 3)
	Len(t, map[string]string{"1": "13"}, 1)
	Len(t, "hello", 5)
}

func TestAssert_LenFails(t *testing.T) {
	m := &mockT{}

	Len(m, []string{"1"}, 2)

	True(t, m.failed)
}
