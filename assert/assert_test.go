package assert

import (
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Drink string `json:"drink"`
}

func TestApiTest_Assert_AssertEquals_StringValue_WithMessage(t *testing.T) {
	Equal(t, "OneString", "OneString", "Should be equal")
}

func TestApiTest_Assert_AssertEquals_IntValue_WithoutMessage(t *testing.T) {
	Equal(t, 420, 420)
}

func TestApiTest_Assert_ObjectsAreEqual(t *testing.T) {
	if !ObjectsAreEqual(420, 420) {
		t.Fatalf("Objects should have been equal")
	}
}

func TestApiTest_Assert_ObjectsAreEqual_ExpectFalse(t *testing.T) {
	if ObjectsAreEqual(420, 421) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestApiTest_Assert_ObjectsAreEqual_MissmatchedType(t *testing.T) {
	if ObjectsAreEqual(420, testItem{"Tom", "Beer"}) {
		t.Fatalf("Objects should not have been equal")
	}
}

func TestApiTest_Assert_JsonEqual(t *testing.T) {
	jsonA := `{"name":"Tom","Drink":"Beer"}`
	jsonB := `{"name":"Tom","Drink":"Beer"}`

	JsonEqual(t, jsonA, jsonB)
}
