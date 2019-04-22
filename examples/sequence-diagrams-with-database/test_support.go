package main

import (
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jmoiron/sqlx"
	"github.com/steinfletcher/apitest"
)

const dbAddr = "host=localhost port=5432 user=postgres password=postgres dbname=apitest sslmode=disable"

func RecordingHook(db DB) apitest.RecorderHook {
	return func(recorder *apitest.Recorder) {
		if v, ok := db.(*recordingDB); ok {
			v.recorder = recorder
		}
	}
}

type recordingDB struct {
	*sqlx.DB
	recorder   *apitest.Recorder
	sourceName string
}

func NewRecordingDB() DB {
	conn, err := sqlx.Connect("postgres", dbAddr)
	if err != nil {
		panic(err)
	}
	return &recordingDB{conn, nil, ""}
}

func (r *recordingDB) Get(dest interface{}, query string, args interface{}) error {
	r.recorder.AddMessageRequest(apitest.MessageRequest{
		Source:    r.sourceName,
		Target:    "database",
		Header:    "SQL Query",
		Body:      query,
		Timestamp: time.Now().UTC(),
	})

	err := r.DB.Get(dest, query, args)

	var body string
	if err != nil {
		body = err.Error()
	} else {
		body = spew.Sprintf("%v", dest)
	}

	r.recorder.AddMessageResponse(apitest.MessageResponse{
		Source:    "database",
		Target:    r.sourceName,
		Header:    "SQL Result",
		Body:      body,
		Timestamp: time.Now().UTC(),
	})

	return err
}

func DBSetup(setup func(db *sqlx.DB)) *sqlx.DB {
	db, err := sqlx.Connect("postgres", dbAddr)
	if err != nil {
		panic(err)
	}
	setup(db)
	return db
}
