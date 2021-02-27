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

// This test requires a mysql database to run

var recorder *apitest.Recorder

func init() {
	recorder = apitest.NewTestRecorder()

	// Wrap your database driver of choice with a recorder
	// and register it so you can use it later
	wrappedDriver := apitestdb.WrapWithRecorder("mysql", recorder)
	sql.Register("wrappedMysql", wrappedDriver)
}

func TestGetUser_With_Default_Report_Formatter(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
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
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
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
	dsn := os.Getenv("MYSQL_DSN")

	// Connect using the previously registered driver
	testDB, err := sqlx.Connect("wrappedMysql", dsn)
	if err != nil {
		panic(err)
	}

	// You can also use the WrapConnectorWithRecorder method
	// without having to register a new database driver
	//
	// cfg, err := mysql.ParseDSN(dsn)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// connector, err := mysql.NewConnector(cfg)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// wrappedConnector := apitestdb.WrapConnectorWithRecorder(connector, "mysql", recorder)
	// testDB := sqlx.NewDb(sql.OpenDB(wrappedConnector), "mysql")

	app := newApp(testDB)

	return apitest.New(name).
		Recorder(recorder).
		Report(apitest.SequenceDiagram()).
		Handler(app.Router)
}
