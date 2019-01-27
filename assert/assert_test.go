package assert

import (
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Drink string `json:"drink"`
}

func TestAssert_AssertEquals_StringValue_WithMessage(t *testing.T) {
	Equal(t, "OneString", "OneString", "Should be equal")
}

func TestAssert_AssertEquals_IntValue_WithoutMessage(t *testing.T) {
	Equal(t, 420, 420)
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

func TestAssert_JsonEqual(t *testing.T) {
	jsonA := `{"name":"Tom","Drink":"Beer"}`
	jsonB := `{"name":"Tom","Drink":"Beer"}`

	JsonEqual(t, jsonA, jsonB)
}

func TestAssert_True(t *testing.T) {
	True(t, true)
}

func TestAssert_False(t *testing.T) {
	False(t, false)
}

func TestAssert_Len(t *testing.T) {
	Len(t, []string{}, 0)
	Len(t, []string{"1"}, 1)
	Len(t, []string{"1", "4", "51"}, 3)
	Len(t, map[string]string{"1": "13"}, 1)
	Len(t, "hello", 5)
}
