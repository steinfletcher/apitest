package apitest

import "testing"

func TestCopyHttpResponse_Nil(t *testing.T) {
	if copyHttpResponse(nil) != nil {
		t.Fatal("expected a nil response to copy as nil")
	}
}
