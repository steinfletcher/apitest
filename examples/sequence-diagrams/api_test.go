package main

import (
	"github.com/steinfletcher/apitest"
	"net/http"
	"testing"
)

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	apitest.New("gets the user 1").
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Get("/user").
		Host("user-service").
		Query("name", "jan").
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(`{"name": "jon", "is_contactable": true}`).
		Report()
}

func TestGetUser_With_Default_Report_Formatter_Overriding_Path(t *testing.T) {
	apitest.New("gets the user 2").
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Get("/user").
		Host("user-service").
		Query("name", "jan").
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(`{"name": "jon", "is_contactable": true}`).
		Report(apitest.NewSequenceDiagramFormatter(".sequence-diagrams"))
}

var getPreferencesMock = apitest.NewMock().
	Get("http://preferences/api/preferences/12345").
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

var getUserMock = apitest.NewMock().
	Get("http://users/api/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()
