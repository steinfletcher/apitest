package main

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber"
)

func newApp() *fiber.App {
	app := fiber.New()
	app.Get("/user/:id", getUser)
	return app
}

// example of a simple fiber handler which returns JSON and sets a cookie
func getUser(c *fiber.Ctx) {
	id := c.Params("id")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if id == "1234" {
		c.Cookie(&fiber.Cookie{
			Name:    "CookieForAndy",
			Value:   "Andy",
			Expires: time.Now().Add(24 * time.Hour),
		})
		c.JSON(map[string]string{"id": "1234", "name": "Andy"})
		return
	}

	c.Status(http.StatusNotFound)
}

func main() {
	err := newApp().Listen("localhost:8080")
	if err != nil {
		panic(err)
	}
}
