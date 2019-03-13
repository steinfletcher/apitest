package main

import (
	"errors"
	"github.com/steinfletcher/apitest"
	"net/http"
	"testing"
)

func TestGetUser_Success(t *testing.T) {
	apitest.New().
		Debug().
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Get("/user").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "is_contactable": true}`).
		End()
}

var getPreferencesMock = apitest.NewMock().
	Get("/preferences/12345").
	AddMatcher(func(r *http.Request, mr *apitest.MockRequest) error {
		// Custom matching func for URL Scheme
		if r.URL.Scheme != "http" {
			return errors.New("request did not have 'http' scheme")
		}
		return nil
	}).
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

var getUserMock = apitest.NewMock().
	Get("/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()
