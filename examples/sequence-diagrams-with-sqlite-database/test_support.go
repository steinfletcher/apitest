package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
)

func DBSetup(dsn string, setup func(db *sqlx.DB)) *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	goose.SetDialect("sqlite3")
	errMigration := goose.Up(db.DB, "./migrations")
	if errMigration != nil {
		panic(errMigration)
	}

	setup(db)
	return db
}
