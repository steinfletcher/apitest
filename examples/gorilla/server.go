package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type App struct {
	Router *mux.Router
}

func NewApp() *App {
	router := mux.NewRouter()
	router.HandleFunc("/user/{id}", GetUser()).Methods("GET")
	return &App{Router: router}
}

func (a *App) Start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func GetUser() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		if id == "1234" {
			user := &User{ID: id, Name: "Andy"}
			bytes, _ := json.Marshal(user)
			_, err := w.Write(bytes)
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func main() {
	NewApp().Start()
}
