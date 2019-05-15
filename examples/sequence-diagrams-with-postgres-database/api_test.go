package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
	"github.com/steinfletcher/apitest"
	"github.com/steinfletcher/apitest/examples/sequence-diagrams-with-postgres-database/test"
	_ "github.com/steinfletcher/apitest/examples/sequence-diagrams-with-postgres-database/test"
)

// This test requires a postgres database to run

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	skip(t)

	username := uuid.NewV4().String()[0:7]
	test.DBSetup(func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES ($1, $2)"
		db.MustExec(q, username, true)
	})

	apiTest("gets the user").
		Mocks(getUserMock(username)).
		Get("/user").
		Query("name", username).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		End()
}

func TestPostUser_With_Default_Report_Formatter(t *testing.T) {
	skip(t)

	username := uuid.NewV4().String()[0:7]
	test.DBSetup(func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES ($1, $2)"
		db.MustExec(q, username, true)
	})

	apiTest("creates a user").
		Mocks(postUserMock(username)).
		Post("/user").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		End()
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

func postUserMock(username string) *apitest.Mock {
	return apitest.NewMock().
		Post("http://users/api/user").
		Body(fmt.Sprintf(`{"name": "%s"}`, username)).
		RespondWith().
		Status(http.StatusOK).
		End()
}

func apiTest(name string) *apitest.APITest {
	app := newApp(test.DBConnect())
	return apitest.New(name).
		Recorder(test.Recorder).
		Report(apitest.SequenceDiagram()).
		Handler(app.Router)
}

func skip(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.SkipNow()
	}
}
