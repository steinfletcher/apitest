package graph

import "github.com/steinfletcher/apitest/examples/graphql/graph/model"

//go:generate gqlgen

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	todos []*model.Todo
}
