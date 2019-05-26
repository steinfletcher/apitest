package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	newApp().start()
}

type App struct {
	Router *mux.Router
}

func newApp() *App {
	router := mux.NewRouter()
	router.HandleFunc("/user/search", getUser()).Methods("POST")
	return &App{Router: router}
}

func (a *App) start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User1
		get("http://users/api/user/12345", &user)

		var contactPreferences ContactPreferences1
		get("http://preferences/api/preferences/12345", &contactPreferences)

		response := UserResponse1{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}
		bytes, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
}

type User1 struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ContactPreferences1 struct {
	IsContactable bool `json:"is_contactable"`
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
