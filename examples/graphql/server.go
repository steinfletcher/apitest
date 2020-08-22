package main

import (
	"log"
	"net/http"
	"os"

	"github.com/steinfletcher/apitest/examples/graphql/graph"
)

const defaultPort = "8000"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, graph.NewHandler()))
}
