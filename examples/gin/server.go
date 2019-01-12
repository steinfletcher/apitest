package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type App struct {
	Router *gin.Engine
}

func NewApp() *App {
	router := gin.Default()
	router.GET("/user/:id", GetUser())
	return &App{Router: router}
}

func (a *App) Start() {
	log.Fatal(http.ListenAndServe(":8888", a.Router))
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("TomsFavouriteDrink", "Beer", 0, "/", "here.com", false, false)

		id := c.Param("id")
		if id == "1234" {
			user := &User{ID: id, Name: "Andy"}
			c.JSON(http.StatusOK, user)
			return
		}

		c.AbortWithStatus(http.StatusNotFound)
		return
	}
}

func main() {
	NewApp().Start()
}
