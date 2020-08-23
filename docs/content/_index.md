---
title: Introduction
type: docs
---

# Getting started

## Overview

A simple and extensible testing library for Go. You can use `apitest` to simplify testing of REST services, HTTP handlers and HTTP clients.

Features:

* Mock external http calls
* Render sequence diagrams on test completion
* Extensible - supports various injection points (see {{< relref "docs/configuration#hooks" >}})
* Custom assert functions
* Custom mock matchers
* Various 3rd party addons - jsonpath assertions, css selector assertions, aws integration and more.


## Installation

Using `go get`

```bash
go get -u github.com/steinfletcher/apitest
```

`apitest` follows semantic versioning and is managed using GitHub releases.

## Anatomy of a test

A test consists of three main parts

- [Configuration]({{< relref "/docs/configuration.md" >}}): defines the `http.handler` that will be tested and any specific test configurations, such as mocks, debug mode and reporting
- [Request]({{< relref "/docs/request.md" >}}): defines the test input. This is typically a http request
- [Assertions]({{< relref "/docs/assertions.md" >}}): defines how the application under test should respond. This is typically a http response

```go
func TestHandler(t *testing.T) {
	apitest.New().                              // configuration
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id": "1234", "name": "Andy"}`))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/user/1234").                      // request
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`). // expectations
		Status(http.StatusOK).
		End()
}
```
