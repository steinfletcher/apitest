package test

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
	"github.com/steinfletcher/apitest"
	apitestdb "github.com/steinfletcher/apitest/x/db"
)

const dsn = "host=localhost port=5432 user=postgres password=postgres dbname=apitest sslmode=disable"

var Recorder *apitest.Recorder

func init() {
	Recorder = apitest.NewTestRecorder()
	wrappedDriver := apitestdb.WrapWithRecorder("postgres", Recorder)
	sql.Register("wrappedPostgres", wrappedDriver)
}

func DBSetup(setup func(db *sqlx.DB)) *sqlx.DB {
	d, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		panic(err)
	}

	errMigration := goose.Up(d.DB, "./migrations")
	if errMigration != nil {
		panic(errMigration)
	}

	setup(d)
	return d
}

func DBConnect() *sqlx.DB {
	testDB, err := sqlx.Connect("wrappedPostgres", dsn)
	if err != nil {
		panic(err)
	}
	return testDB
}
