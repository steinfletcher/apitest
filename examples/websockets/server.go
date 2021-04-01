package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func WsHttpHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := &websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func main() {
	http.HandleFunc("/", WsHttpHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
