package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	newApp(NewDB()).start()
}

type App struct {
	Router *mux.Router
	DB     DB
}

func newApp(db DB) *App {
	router := mux.NewRouter()
	router.HandleFunc("/user", getUser(db)).Methods("GET")
	return &App{Router: router, DB: db}
}

func (a *App) start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func getUser(db DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		get(fmt.Sprintf("http://users/api/user?id=%s", r.URL.Query()["name"]), &user)

		var isContactable bool
		err := db.Get(&isContactable,
			"SELECT is_contactable from users where username=$1 LIMIT 1", user.Name)
		if err != nil {
			panic(err)
		}

		response := UserResponse1{
			Name:          user.Name,
			IsContactable: isContactable,
		}

		bytes, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
}

type User struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type UserResponse1 struct {
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
