# Request

[![GoDoc](https://godoc.org/github.com/steinfletcher/apitest?status.svg)](https://godoc.org/github.com/steinfletcher/apitest#Request)

To configure the initial request into the system under test, you can specify request parameters such as the http method, url, headers and cookies.


## Creating a request

```go
apitest.New().
	Handler(handler).
	Method(http.MethodGet).
	URL("/user/12345")
```

This is quite verbose, so there are some shortcuts defined for the common http verbs that wrap up the `Method` and `URL` functions. The example can be more concisely defined as

```go
apitest.New().
	Handler(handler).
	Get("/user/12345")
```

## Query parameters

There are multiple ways to specify query parameters. These approaches are chainable.

### params

```go
Query("param", "value")
```

### map

```go
QueryParams(map[string]string{"param1": "value1", "param2": "value2"})
```

### collection

```go
QueryCollection(map[string][]string{"a": {"1", "2"}})
```

Providing `{"a": {"1", "2"}` results in parameters encoded as `a=1&a=2`. 

### custom

If none of the above approaches is suitable, you can defined a request interceptor and implement custom logic.

```go
apitest.New().
	Handler(handler).
	Intercept(func(req *http.Request) {
		req.URL.RawQuery = "a[]=xxx&a[]=yyy"
	}).
	Get("/path")
```

## Headers

There are multiple ways to specify http request headers. These approaches are chainable.

### params

```go
Header("name", "value")
```

### map

```go
Headers(map[string]string{"name1": "value1", "name2": "value2"})
```

## Cookies

There are multiple ways to specify http request cookies. These approaches are chainable.

### short form

```go
Cookie("name", "value")
```

### struct

`Cookies` is a variadic function that can be used to take a variable amount of cookies defined as a struct

```go
Cookies(apitest.NewCookie("name").
	Value("value").
	Path("/user").
	Domain("example.com"))
```

The underlying fields of this struct are all pointer types. This allows the assertion library to ignore fields that are not defined in the struct.

## Body

There are two methods to set the request body - `Body` and `JSON`. `Body` will be copied to the raw request and wrapped in an `io.Reader`.

```go
Post("/message").
Body("hello")
``` 
 
`JSON` does the same and copies the provided data to the body, but the `JSON` method also sets the content type to `application/json`.

```go
Post("/chat").
JSON(`{"message": "hi"}`)
```

If you want to define other content types set the body using `Body(data)` and the header using `Header("Content-Type", "application/xml")`.

```go
Post("/path").
Body("<html>content</html>").
Header("Content-Type", "text/html")
```

## Basic auth

A helper method is provided to add preemptive basic authentication to the request. 

```go
Get("/path").
BasicAuth("username", "password").
```
