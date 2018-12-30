***This repo is experimental and isn't ready to be consumed***

# api-test

[![Build Status](https://travis-ci.org/steinfletcher/api-test.svg?branch=master)](https://travis-ci.org/steinfletcher/api-test) [![Coverage Status](https://coveralls.io/repos/github/steinfletcher/api-test/badge.svg?branch=master)](https://coveralls.io/github/steinfletcher/api-test?branch=master)

Simple behavioural (black box) api testing library. 

_In black box tests the internal structure of the app is not know by the tests. Data is input by calling the rest endpoints with a http client and the outputs are expected to meet certain conditions._

## Installation

     go get -u github.com/steinfletcher/api-test

## Examples

JSON body matcher

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Tate"}`).
		Status(http.StatusCreated).
		End()
}
```

JSONPath body matcher. Given the response is `{"a": 12345, "b": [{"key": "c", "value": "result"}]}`

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		Expect(t).
		JSONPath(`$.b[? @.key=="c"].value`, func(values interface{}) {
			assert.Contains(t, values, "result")
		}).
		End()
}
```

Custom assert functions.

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		Expect(t).
		Assert(func(res *http.Response, req *http.Request) {
			assert.Equal(t, http.StatusOK, res.StatusCode)
		}).
		End()
}
```

Assert cookies

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(map[string]string{
			"ABC": "12345",
			"DEF": "67890",
		}).
		CookiePresent("Session-Token").
		HttpCookies([]http.Cookie{
			{Name: "HIJ", Value: "12345"},
		}).
		End()
}
```

Assert headers

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{"ABC": "12345"}).
		End()
}
```

Provide basic auth in the request

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		BasicAuth("username:password").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

Provide cookies in the request

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		Cookies(map[string]string{"Cookie1": "Yummy"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

Provide headers in the request

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Delete("/hello").
		Headers(map[string]string{"My-Header": "12345"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

Provide query parameters in the request

```go
func TestGetUser(t *testing.T) {
	apitest.New(handler).
		Get("/hello").
		Query(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```
