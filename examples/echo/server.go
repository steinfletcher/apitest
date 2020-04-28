package main

import (
	"net/http"

	"github.com/labstack/echo"
)

func newApp() *echo.Echo {
	app := echo.New()
	app.GET("/user/:id", getUser)
	return app
}

// example of a simple echo handler which returns JSON and sets a cookie
func getUser(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return echo.ErrBadRequest
	}

	if id == "1234" {
		ctx.SetCookie(&http.Cookie{
			Name:  "CookieForAndy",
			Value: "Andy",
		})
		return ctx.JSON(200, map[string]string{"id": "1234", "name": "Andy"})
	}

	return ctx.NoContent(404)
}

func main() {
	err := newApp().Start("localhost:8080")
	if err != nil {
		panic(err)
	}
}
