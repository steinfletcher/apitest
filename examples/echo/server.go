package main

import (
	"net/http"

	"github.com/labstack/echo"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type application struct {
	app *echo.Echo
}

func newApp() *application {
	app := echo.New()

	app.GET("/user/:id", getUser)

	return &application{
		app: app,
	}
}

func (a *application) start() {
	a.app.Logger.Fatal(a.app.Start("localhost:1323"))
}

func getUser(ctx echo.Context) error {
	ctx.SetCookie(&http.Cookie{
		Name:  "TomsFavouriteDrink",
		Value: "Beer",
		Path:  "/",
	})

	id := ctx.Param("id")

	if id == "1234" {
		return ctx.JSON(200, &user{ID: id, Name: "Andy"})
	}
	return ctx.NoContent(404)
}

func main() {
	newApp().start()
}
