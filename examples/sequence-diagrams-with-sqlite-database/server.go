package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose"
)

func main() {
	dsn := os.Getenv("SQLITE_DSN")
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	goose.SetDialect("sqlite3")
	errMigration := goose.Up(db.DB, "./migrations")
	if errMigration != nil {
		panic(errMigration)
	}

	newApp(db).start()
}

type App struct {
	Router *mux.Router
	DB     *sqlx.DB
}

func newApp(db *sqlx.DB) *App {
	router := mux.NewRouter()
	router.HandleFunc("/user", getUser(db)).Methods("GET")
	router.HandleFunc("/user", postUser(db)).Methods("POST")
	return &App{Router: router, DB: db}
}

func (a *App) start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func getUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var externalUser ExternalUser
		get(fmt.Sprintf("http://users/api/user?id=%s", r.URL.Query()["name"]), &externalUser)

		var isContactable bool
		err := db.Get(&isContactable, "SELECT is_contactable FROM users WHERE username=? AND is_contactable=? LIMIT 1", externalUser.Name, true)
		if err != nil {
			panic(err)
		}

		response := User{
			Name:          externalUser.Name,
			IsContactable: isContactable,
		}

		bytes, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
}

func postUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(body, &user)
		if err != nil {
			panic(err)
		}

		post("http://users/api/user", ExternalUser{Name: user.Name})

		tx := db.MustBegin()
		_, err = tx.Exec("INSERT INTO users (username, is_contactable) VALUES (?, ?)", user.Name, user.IsContactable)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error inserting user", http.StatusInternalServerError)
			return
		}
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

type ExternalUser struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type User struct {
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

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(bytes, response)
	if err != nil {
		panic(err)
	}
}

func post(path string, user ExternalUser) {
	requestBytes, _ := json.Marshal(user)

	res, err := http.Post(path, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		panic(err)
	}

	if !(res.StatusCode >= http.StatusOK && res.StatusCode < 400) {
		panic(fmt.Sprintf("unexpected status code=%d", res.StatusCode))
	}
}
