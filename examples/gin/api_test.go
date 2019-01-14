package main

import (
	"github.com/steinfletcher/api-test"
	"github.com/steinfletcher/api-test-jsonpath"
	"net/http"
	"testing"
)

func TestGetUser_CookieMatching(t *testing.T) {
	apitest.New().
		Handler(NewApp().Router).
		Get("/user/1234").
		Expect(t).
		Cookies(apitest.ExpectedCookie("TomsFavouriteDrink").
			Value("Beer").
			Path("/")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success(t *testing.T) {
	apitest.New().
		Handler(NewApp().Router).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success_JSONPath(t *testing.T) {
	apitest.New().
		Handler(NewApp().Router).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_NotFound(t *testing.T) {
	apitest.New().
		Handler(NewApp().Router).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}
