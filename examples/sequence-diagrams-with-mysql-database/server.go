package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		panic(err)
	}

	err = goose.SetDialect("mysql")
	if err != nil {
		panic(err)
	}
	errMigration := goose.Up(db.DB, "./migrations")
	if errMigration != nil {
		panic(errMigration)
	}

	newApp(db).start()
}

type app struct {
	Router *mux.Router
	DB     *sqlx.DB
}

func newApp(db *sqlx.DB) *app {
	router := mux.NewRouter()
	router.HandleFunc("/user", getUser(db)).Methods("GET")
	router.HandleFunc("/user", postUser(db)).Methods("POST")
	return &app{Router: router, DB: db}
}

func (a *app) start() {
	log.Fatal(http.ListenAndServe("localhost:8888", a.Router))
}

func getUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var externalUser externalUser
		get(fmt.Sprintf("http://users/api/user?id=%s", r.URL.Query()["name"]), &externalUser)

		var isContactable bool
		err := db.Get(&isContactable, "SELECT is_contactable FROM users WHERE username=? AND is_contactable=? LIMIT 1", externalUser.Name, true)
		if err != nil {
			panic(err)
		}

		response := user{
			Name:          externalUser.Name,
			IsContactable: isContactable,
		}

		b, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func postUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user user
		body, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(body, &user)
		if err != nil {
			panic(err)
		}

		post("http://users/api/user", externalUser{Name: user.Name})

		tx := db.MustBegin()
		_, err = tx.Exec("INSERT INTO users (username, is_contactable) VALUES (?, ?)", user.Name, user.IsContactable)
		if err != nil {
			err := tx.Rollback()
			if err != nil {
				panic(err)
			}
			http.Error(w, "Error inserting user", http.StatusInternalServerError)
			return
		}
		err = tx.Commit()
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

type externalUser struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type user struct {
	Name          string `json:"name"`
	IsContactable bool   `json:"is_contactable"`
}

func get(path string, response interface{}) {
	res, err := http.Get(path)
	if err != nil {
		panic(err)
	}

	if !(res.StatusCode >= http.StatusOK && res.StatusCode < 400) {
		panic(fmt.Sprintf("unexpected status code=%d", res.StatusCode))
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(bytes, response)
	if err != nil {
		panic(err)
	}
}

func post(path string, user externalUser) {
	requestBytes, _ := json.Marshal(user)

	res, err := http.Post(path, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		panic(err)
	}

	if !(res.StatusCode >= http.StatusOK && res.StatusCode < 400) {
		panic(fmt.Sprintf("unexpected status code=%d", res.StatusCode))
	}
}
