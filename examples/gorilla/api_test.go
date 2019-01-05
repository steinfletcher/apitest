package main

import (
	"github.com/steinfletcher/api-test"
	"net/http"
	"testing"
)

func TestGetUser_Success(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUser_NotFound(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}
