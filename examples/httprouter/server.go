package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func newRouter() *httprouter.Router {
	router := httprouter.New()
	router.GET("/user/:id", getUser)
	return router
}

func getUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if id == "1234" {
		http.SetCookie(w, &http.Cookie{
			Name:  "CookieForAndy",
			Value: "Andy",
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]string{"id": "1234", "name": "Andy"}
		bytes, _ := json.Marshal(response)
		if _, err := w.Write(bytes); err != nil {
			panic(err)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
	return
}

func main() {
	router := newRouter()
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
