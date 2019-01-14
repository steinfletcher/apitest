package main

import (
	"github.com/kataras/iris"
	"log"
	"net/http"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type App struct {
	iris *iris.Application
}

func newApp() *App {
	app := iris.Default()
	app.Get("/user/{id}", getUser)
	if err := app.Build(); err != nil {
		panic(err)
	}
	return &App{
		iris: app,
	}
}

func (a *App) start() {
	log.Fatal(a.iris.Run(iris.Addr(":8080")))
}

func getUser(ctx iris.Context) {
	ctx.SetCookie(&http.Cookie{
		Name:  "TomsFavouriteDrink",
		Value: "Beer",
		Path:  "/",
	})

	id := ctx.Params().Get("id")

	if id == "1234" {
		_, e := ctx.JSON(&User{ID: id, Name: "Andy"})
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
