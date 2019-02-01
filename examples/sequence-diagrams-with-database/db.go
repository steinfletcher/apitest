package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB interface {
	Get(dest interface{}, query string, args interface{}) error
}

type db struct {
	*sqlx.DB
}

func NewDB() DB {
	dsn := fmt.Sprintf(
		"host=localhost port=5432 user=postgres password=postgres dbname=apitest sslmode=disable",
	)
	conn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		panic(err)
	}

	return &db{conn}
}

func (r db) Get(dest interface{}, query string, args interface{}) error {
	return r.DB.Get(dest, query, args)
}
