package main

import (
	"net/http"

	"github.com/labstack/echo"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type App struct {
	app *echo.Echo
}

func newApp() *App {
	app := echo.New()

	app.GET("/user/:id", getUser)

	return &App{
		app: app,
	}
}

func (a *App) start() {
	a.app.Logger.Fatal(a.app.Start(":1323"))
}

func getUser(ctx echo.Context) error {
	ctx.SetCookie(&http.Cookie{
		Name:  "TomsFavouriteDrink",
		Value: "Beer",
		Path:  "/",
	})

	id := ctx.Param("id")

	if id == "1234" {
		return ctx.JSON(200, &User{ID: id, Name: "Andy"})
	} else {
		return ctx.NoContent(404)
	}
}

func main() {
	newApp().start()
}
