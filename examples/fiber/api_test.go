package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
)

func TestGetUser_CookieMatching(t *testing.T) {
	apitest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Cookies(apitest.NewCookie("CookieForAndy").Value("Andy")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success(t *testing.T) {
	apitest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success_JSONPath(t *testing.T) {
	apitest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_NotFound(t *testing.T) {
	apitest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}

func FiberToHandlerFunc(app *fiber.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := app.Test(r)
		if err != nil {
			panic(err)
		}

		// copy headers
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)

		if _, err := io.Copy(w, resp.Body); err != nil {
			panic(err)
		}
	}
}
