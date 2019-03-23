# Assertions

`apitest` provides several mechanisms to verify the response data. If none of these suits your needs you can provide custom `Assert` functions. 

After defining the request, you must call `Expect(t)` to begin defining assertions, like so

Example:

```go
New().
    Handler(handler).
    Get("/user").
    Expect(t).
    Body(`{"name": "yuki"}`).
    Status(http.StatusCreated).
    End()
```

## Status Code

```go
Expect(t).
Status(http.StatusOK)
```

## Body

To match the HTTP response body pass a string into the `Body` method.

```go
Expect(t).
Body(`{"param": "value"}`)
```

The assertion library checks if the content is `JSON` and if so performs the assertion using `testify's` [assert.JSONEq](https://godoc.org/github.com/stretchr/testify/assert#JSONEq) method. If the content is not `JSON`, `testify's` [assert.Equal](https://godoc.org/github.com/stretchr/testify/assert#Equal) method is used 

## Cookies

Example:

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(apitest.Cookie"ABC").Value("12345")).
		CookiePresent("Session-Token").
		CookieNotPresent("XXX").
		End()
}

```

### Short form

The simplest way to assert a response cookie is to provide the cookie name and value as parameters to the `Cookie` method.

```go
Cookie("name", "value")
```

### Struct

`Cookies` is a variadic function that can be used to take a variable amount of cookies defined as a struct

```go
Cookies(apitest.NewCookie("name").
	Value("value").
	Path("/user").
	Domain("example.com"))
```

The underlying fields of this struct are all pointer types. This allows the assertion library to ignore fields that are not defined in the struct.

### Present

Sometimes an application will generate a cookie with a dynamic value. If you do not need to assert on the value, use the `CookiePresent` method which will only assert that a cookie has been set with a given key.

```go
CookiePresent("Session-Token")
```

`apitest` keeps a slice of cookies internally so you can invoke this method many times to assert on multiple cookies.

### Not Present

This is the opposite behaviour of `CookiePresent` and is used to assert that a cookie with a given name is not present in the response.

```go
CookieNotPresent("Session-Token")
```

`apitest` keeps a slice of cookies internally so you can invoke this method many times to assert on multiple cookies.

## Headers

There are two ways to specify HTTP response headers. The following approaches are chainable.

### params

```go
Header("name", "value")
```

### map

```go
Headers(map[string]string{"name1": "value1", "name2": "value2"})
```

*Note*: headers are stored internally in `apitest` in their canonical form. For example, the canonical key for "accept-encoding" is "Accept-Encoding".
