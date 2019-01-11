**This library is on Version 0 and we won't guarantee backwards compatible API changes until we go to version 1**

# api-test

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/023e062d720847e08c1065cbb65a4068)](https://app.codacy.com/app/steinfletcher/api-test?utm_source=github.com&utm_medium=referral&utm_content=steinfletcher/api-test&utm_campaign=Badge_Grade_Dashboard)
[![Build Status](https://travis-ci.org/steinfletcher/api-test.svg?branch=master)](https://travis-ci.org/steinfletcher/api-test) [![Coverage Status](https://coveralls.io/repos/github/steinfletcher/api-test/badge.svg?branch=master)](https://coveralls.io/github/steinfletcher/api-test?branch=master)

Simple behavioural (black box) api testing library. 

In black box tests the internal structure of the app is not know by the tests. Data is input to the system and the outputs are expected to meet certain conditions.

Check the godoc [here](https://godoc.org/github.com/steinfletcher/api-test).

## Installation

```bash
go get -u github.com/steinfletcher/api-test
```

## Examples

### Framework and library integration examples

| Example                                                                           | Comment                             |
| --------------------------------------------------------------------------------- | ----------------------------------- |
| [gin](https://github.com/steinfletcher/api-test/tree/master/examples/gin)         | popular martini-like web framework  |
| [gorilla](https://github.com/steinfletcher/api-test/tree/master/examples/gorilla) | the gorilla web toolkit             |

### Companion libraries

| Library                                                        | Comment                   |
| -------------------------------------------------------------- | ------------------------- |
| [JSONPath](https://github.com/steinfletcher/api-test-jsonpath) | JSONPath assertion addons |

### Code snippets

#### JSON body matcher

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Tate"}`).
		Status(http.StatusCreated).
		End()
}
```

#### JSONPath

For asserting on parts of the response body JSONPath may be used. A separate module must be installed which provides these assertions - `go get -u github.com/steinfletcher/api-test-jsonpath`. This is packaged separately to keep this library dependency free.

Given the response is `{"a": 12345, "b": [{"key": "c", "value": "result"}]}`

```go
	apitest.New().
	Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Contains(`$.b[? @.key=="c"].value`, "result")).
		End()
```

and `jsonpath.Equals` checks for value equality

```go
	New(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$.a`, float64(12345))).
		End()
```

#### Custom assert functions

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(func(res *http.Response, req *http.Request) {
			assert.Equal(t, http.StatusOK, res.StatusCode)
		}).
		End()
}
```

#### Assert cookies

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(map[string]string{
			"ABC": "12345",
			"DEF": "67890",
		}).
		CookiePresent("Session-Token").
		CookieNotPresent("XXX").
		HttpCookies([]http.Cookie{
			{Name: "HIJ", Value: "12345"},
		}).
		End()
}
```

#### Assert headers

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{"ABC": "12345"}).
		End()
}
```

#### Provide basic auth in the request

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		BasicAuth("username:password").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide cookies in the request

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		Cookies(map[string]string{"Cookie1": "Yummy"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide headers in the request

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Delete("/hello").
		Headers(map[string]string{"My-Header": "12345"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide query parameters in the request

`Query` can be used in combination with `QueryCollection`

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		Query(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide query parameters in collection format in the request

Providing `{"a": {"b", "c", "d"}` results in parameters encoded as `a=b&a=c&a=d`.
`QueryCollection` can be used in combination with `Query`

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{"a": {"b", "c", "d"}}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Capture the request and response data

```go
func TestApi(t *testing.T) {
	apitest.New().
		Observe(func(res *http.Response, req *http.Request) {
    	    // do something with res and req
    	}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

one usage for this might be debug logging to the console. The provided `DumpHttp` function does this automatically

```go
func TestApi(t *testing.T) {
	apitest.New().
		Observe(apitest.DumpHttp).
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusCreated).
		End()
}
```

#### Intercept the request

This is useful for mutating the request before it is sent to the system under test.

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Intercept(func(req *http.Request) {
			req.URL.RawQuery = "a[]=xxx&a[]=yyy"
		}).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```
