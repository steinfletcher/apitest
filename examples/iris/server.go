package main

import (
	"log"
	"net/http"

	"github.com/kataras/iris"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type application struct {
	iris *iris.Application
}

func newApp() *application {
	app := iris.Default()
	app.Get("/user/{id}", getUser)
	if err := app.Build(); err != nil {
		panic(err)
	}
	return &application{
		iris: app,
	}
}

func (a *application) start() {
	log.Fatal(a.iris.Run(iris.Addr("localhost:8080")))
}

func getUser(ctx iris.Context) {
	ctx.SetCookie(&http.Cookie{
		Name:  "TomsFavouriteDrink",
		Value: "Beer",
		Path:  "/",
	})

	id := ctx.Params().Get("id")

	if id == "1234" {
		_, e := ctx.JSON(&user{ID: id, Name: "Andy"})
		if e != nil {
			panic(e)
		}
	} else {
		ctx.NotFound()
	}
}

func main() {
	newApp().start()
}
