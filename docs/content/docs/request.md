# Request

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

which results in parameters encoded as `a=1&a=2`. 

### custom

If none of the above approaches are suitable, you can define a request interceptor and implement custom logic.

```go
apitest.New().
	Handler(handler).
	Intercept(func(req *http.Request) {
		req.URL.RawQuery = "a[]=xxx&a[]=yyy"
	}).
	Get("/path")
```

## Headers

There are multiple ways to specify http request headers. The following approaches are chainable.

### params

```go
Header("name", "value")
```

### map

```go
Headers(map[string]string{"name1": "value1", "name2": "value2"})
```

## URL encoded form

There are multiple ways to create a URL encoded form body in the request. The following approaches are chainable.

### short form

```go
FormData("name", "value")
```

### multiple values

`FormData` is a variadic function that can be used to take a variable amount of values for the same key.

```go
FormData("name", "value1", "value2")
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

## GraphQL

The following helpers simplify building GraphQL requests.

```go
Post("/graphql").
GraphQLQuery(`query { todos { text } }`).
```

```go
Post("/graphql").
GraphQLRequest(apitest.GraphQLRequestBody{
	Query: "query someTest($arg: String!) { test(who: $arg) }",
	Variables: map[string]interface{}{
		"arg": "myArg",
	},
	OperationName: "myOperation",
}).
```

## Basic auth

A helper method is provided to add preemptive basic authentication to the request. 

```go
Get("/path").
BasicAuth("username", "password").
```

## Intercept

You can intercept the request before it is sent to the system under test. This can be useful for setting global http headers and custom query parameters.
See [custom cookies]({{< relref "/docs/request.md#custom" >}})

```go
apitest.New(name).
	Handler(router).
	Intercept(func(request *http.Request) {
		request.Header.Set("Authorization", "1234567890")
	})
```