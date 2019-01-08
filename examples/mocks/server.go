package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type App struct {
	Router *mux.Router
}

func newApp() *App {
	router := mux.NewRouter()
	router.HandleFunc("/user", getUser()).Methods("GET")
	return &App{Router: router}
}

func (a *App) start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		get("/user/12345", &user)

		var contactPreferences ContactPreferences
		get("/preferences/12345", &contactPreferences)

		response := UserResponse{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}
		bytes, _ := json.Marshal(response)
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
}

type User struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ContactPreferences struct {
	IsContactable bool `json:"is_contactable"`
}

type UserResponse struct {
	Name          string `json:"name"`
	IsContactable bool   `json:"is_contactable"`
}

func main() {
	newApp().start()
}

func get(path string, response interface{}) {
	res, err := http.Get(fmt.Sprintf("http://localhost:8080%s", path))
	if err != nil || res.StatusCode != http.StatusOK {
		panic(err)
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
