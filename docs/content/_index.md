---
title: Introduction
type: docs
---

# Getting started

## Overview

apitest is a simple and extensible behavioural testing library written in golang. It supports mocking external http calls and renders sequence diagrams on test completion.

In behavioural tests the internal structure of the application is not known by the tests. Data is input to the system under test (SUT) and the outputs are expected to meet certain conditions.

## Installation

Using `go get`

```bash
go get -u github.com/steinfletcher/apitest
```

Using `dep`

```bash
dep ensure -add github.com/steinfletcher/apitest
```

apitest is tested against Go `1.11.x` and `stable` and follows semantic versioning managed through GitHub releases.

## Anatomy of a test

A test consists of three main parts

- [Configuration](http://todo): defines the `http.handler` that will be tested and any specific test configurations
- [Request](http://todo): defines the test input. This is typically a http request
- [Expectations](http://todo): defines how the application under test should respond. This is typically a http response

```go
func TestGetUser(t *testing.T) {
	apitest.New().                              // configuration
		Handler(newApp().app).
		Get("/user/1234").                      // request
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`). // expectations
		Status(http.StatusOK).
		End()
}
```
