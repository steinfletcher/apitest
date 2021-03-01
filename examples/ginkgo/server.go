package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type application struct {
	Router *mux.Router
}

func NewApp() *application {
	router := mux.NewRouter()
	router.HandleFunc("/user/{id}", getUser()).Methods("GET")
	return &application{Router: router}
}

func (a *application) start() {
	log.Fatal(http.ListenAndServe("localhost:8888", a.Router))
}

func getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		http.SetCookie(w, &http.Cookie{
			Name:  "TomsFavouriteDrink",
			Value: "Beer",
			Path:  "/",
		})

		if id == "1234" {
			user := &user{ID: id, Name: "Andy"}
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
	}
}

func main() {
	NewApp().start()
}
