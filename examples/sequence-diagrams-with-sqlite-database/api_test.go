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

var recorder *apitest.Recorder

func init() {
	recorder = apitest.NewTestRecorder()
	wrappedDriver := apitestdb.WrapWithRecorder("sqlite3", recorder)
	sql.Register("wrappedSqlite", wrappedDriver)
}

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	dsn := os.Getenv("SQLITE_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	apiTest("gets the user").
		Mocks(getUserMock(username)).
		Get("/some-really-long-path-so-we-can-observe-truncation-here-whey").
		Query("name", username).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		End()
}

func TestPostUser_With_Default_Report_Formatter(t *testing.T) {
	dsn := os.Getenv("SQLITE_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	apiTest("creates a user").
		Mocks(postUserMock(username)).
		Post("/user").
		JSON(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
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
	dsn := os.Getenv("SQLITE_DSN")
	testDB, err := sqlx.Connect("wrappedSqlite", dsn)
	if err != nil {
		panic(err)
	}

	app := newApp(testDB)

	return apitest.New(name).
		Recorder(recorder).
		Report(apitest.SequenceDiagram()).
		Handler(app.Router)
}
