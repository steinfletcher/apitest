package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
	"github.com/steinfletcher/apitest"
	apitestdb "github.com/steinfletcher/apitest/x/db"
)

// This test requires a postgres database to run

var recorder *apitest.Recorder

func init() {
	recorder = apitest.NewTestRecorder()
	wrappedDriver := apitestdb.WrapWithRecorder("postgres", recorder)
	sql.Register("wrappedPostgres", wrappedDriver)
}

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES ($1, $2)"
		db.MustExec(q, username, true)
	})

	apiTest("gets the user").
		Debug().
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
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES ($1, $2)"
		db.MustExec(q, username, true)
	})

	apiTest("creates a user").
		Debug().
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
	dsn := os.Getenv("POSTGRES_DSN")
	testDB, err := sqlx.Connect("wrappedPostgres", dsn)
	if err != nil {
		panic(err)
	}

	app := newApp(testDB)

	return apitest.New(name).
		Recorder(recorder).
		Report(apitest.SequenceDiagram()).
		Handler(app.Router)
}
