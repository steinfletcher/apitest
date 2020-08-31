package main_test

import (
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/steinfletcher/apitest/examples/graphql/graph"
	"net/http"
	"testing"
)

func TestQuery_Empty(t *testing.T) {
	apitest.New().
		Handler(graph.NewHandler()).
		Post("/query").
		GraphQLQuery(`query {
			todos {
				text
				done
				user {
					name
				}
			}
		}`).
		Expect(t).
		Status(http.StatusOK).
		Body(`{
		  "data": {
			"todos": []
		  }
		}`).
		End()
}

func TestQuery_WithTodo(t *testing.T) {
	handler := graph.NewHandler()

	apitest.New().
		Handler(handler).
		Post("/query").
		JSON(`{"query": "mutation { createTodo(input:{text:\"todo\", userId:\"4\"}) { user { id } text done } }"}`).
		Expect(t).
		Status(http.StatusOK).
		Assert(jsonpath.Equal("$.data.createTodo.user.id", "4")).
		End()

	apitest.New().
		Handler(handler).
		Post("/query").
		GraphQLQuery("query { todos { text done user { name } } }").
		Expect(t).
		Status(http.StatusOK).
		Body(`{
		  "data": {
			"todos": [
			  {
				"text": "todo",
				"done": false,
				"user": {
				  "name": "user"
				}
			  }
			]
		  }
		}`).
		End()
}
