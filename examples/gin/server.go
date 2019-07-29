package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type application struct {
	Router *gin.Engine
}

func newApp() *application {
	router := gin.Default()
	router.GET("/user/:id", getUser())
	return &application{Router: router}
}

func (a *application) Start() {
	log.Fatal(http.ListenAndServe("localhost:8888", a.Router))
}

func getUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("TomsFavouriteDrink", "Beer", 0, "/", "here.com", false, false)

		id := c.Param("id")
		if id == "1234" {
			user := &user{ID: id, Name: "Andy"}
			c.JSON(http.StatusOK, user)
			return
		}

		c.AbortWithStatus(http.StatusNotFound)
	}
}

func main() {
	newApp().Start()
}
