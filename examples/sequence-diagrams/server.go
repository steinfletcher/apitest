package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	newApp().start()
}

type app struct {
	Router *mux.Router
}

func newApp() *app {
	router := mux.NewRouter()
	router.HandleFunc("/user/search", getUser()).Methods("POST")
	return &app{Router: router}
}

func (a *app) start() {
	log.Fatal(http.ListenAndServe("localhost:8888", a.Router))
}

func getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user user
		get("http://users/api/user/12345", &user)

		var contactPreferences contactPreferences
		get("http://preferences/api/preferences/12345", &contactPreferences)

		response := userResponse{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}
		bytes, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(bytes); err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

type user struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type contactPreferences struct {
	IsContactable bool `json:"is_contactable"`
}

type userResponse struct {
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
