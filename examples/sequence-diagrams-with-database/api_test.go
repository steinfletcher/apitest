package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/satori/go.uuid"
	"github.com/steinfletcher/apitest"
	"net/http"
	"testing"
)

// This test requires a postgres database to run

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	t.SkipNow()

	testDB := NewRecordingDB()
	app := newApp(testDB)
	username := uuid.NewV4().String()[0:7]

	DBSetup(func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES ('%s', %v)"
		db.MustExec(fmt.Sprintf(q, username, true))
	})

	apitest.New("gets the user").
		Mocks(getUserMock(username)).
		RecorderHook(RecordingHook(testDB)).
		Handler(app.Router).
		Get("/user").
		Query("name", username).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		Report()
}

func getUserMock(username string) *apitest.Mock {
	return apitest.NewMock().
		Get("http://users/api/user").
		Query("id", username).
		RespondWith().
		Body(fmt.Sprintf(`{"name": "%s", "id": "1234"}`, username)).
		Status(http.StatusOK).
		End()
}
