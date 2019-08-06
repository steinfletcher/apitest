# Mocks

In this [example](https://github.com/steinfletcher/apitest/tree/master/examples/mocks) we implement a REST API and mock calls to an external system.

## Why would I use mocks?

It is very common for an application to integrate with an external API. When running tests in the development phase a short feedback loop is desirable and it is important that the tests are repeatable and reproducible. Integrating with the real external API adds unknown factors that often cause tests to break for reasons out of your control.

Mocking external calls improves the stability of the development lifecycle testing phase helping you to ship features with confidence more quickly. This does not replace integration testing. There are no hard rules and the testing strategy will vary from project to project.

## How mocking works

The mocks in `apitest` are heavily inspired by [gock](https://github.com/h2non/gock).

The mock package hijacks the default HTTP transport and implements a custom `RoundTrip` method. If the outgoing HTTP request matches against a collection of defined mocks, the result defined in the mock will be returned to the caller.

## Defining mocks

A mock is defined by calling the `apitest.NewMock()` factory method

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

in the above example, when a HTTP client makes a `GET` request to `http://example.com/user/12345`, then `{"name": "jon"}` is returned in the response body with HTTP status code `200`.

The mock can then be added to the `apitest` configuration section as follows

```go
apitest.New().
    Mocks(getPreferencesMock, getUserMock).
    Handler(newApp().Router).
    Get("/user").
    Expect(t).
    Status(http.StatusOK).
    End()
```

Note that multiple mocks can be defined. Due to FIFO ordering if a request matches more than one mock the first mock matched is used.

## Adding matchers to mocks

You can add matchers for the request headers, cookies, url query parameters and body.

### Header

`Header()` allows you to add a matcher for the header key and value. Regular expressions are also allowed as values.

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    Header("foo", "bar").
    Header("token", "b([a-z]+)z").
    Headers(map[string]string{"name": "John"})
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

You can also require a header to be present (`HeaderPresent()`) or not present (`HeaderNotPresent()`)

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    HeaderPresent("authtoken").
    HeaderNotPresent("requestid").
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

### Query Parameters

`Query()` allows you to add a matcher for the a url query parameter key and value. Regular expressions are also allowed as values.

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    Query("page", "1").
    Query("name", "Jo([a-z]+)n").
    QueryParams(map[string]string{"orderBy": "ASC"}).
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

You can also require a query parameter to be present (`QueryPresent()`) or not present (`QueryNotPresent()`)

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    QueryPresent("page").
    QueryNotPresent("name").
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

### Cookies

`Cookie()` allows you to add a matcher for a cookie name and value.

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    Cookie("sessionid", "1321").
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

You can also require a cookie name to be present (`CookiePresent()`) or not present (`CookieNotPresent()`)

```go
var getUserMock = apitest.NewMock().
    Get("http://example.com/user/12345").
    CookiePresent("trackingid").
    CookieNotPresent("analytics").
    RespondWith().
    Body(`{"name": "jon"}`).
    Status(http.StatusOK).
    End()
```

### Body

`Body()` allows you to add a matcher for the body of the request.

```go
var getUserMock = apitest.NewMock().
    Post("http://example.com/user/12345").
    Body(`{"username": "John"}`).
    RespondWith().
    Status(http.StatusOK).
    End()
```

If you are working with a URL encoded form body, you can use `FormData()` to match a key and value. Regular expressions are also allowed as values.

```go
var getUserMock = apitest.NewMock().
    Post("http://example.com/user/12345").
    FormData("name", "Simon").
    FormData("name", "Jo([a-z]+)n").
    RespondWith().
    Status(http.StatusOK).
    End()
```

You can also require a form body key to be present (`FormDataPresent()`) or not present (`FormDataNotPresent()`)

```go
var getUserMock = apitest.NewMock().
    Post("http://example.com/user/12345").
    FormDataPresent("name").
    FormDataNotPresent("pets").
    RespondWith().
    Status(http.StatusOK).
    End()
```

### Custom matcher

You can write you own custom matcher using `AddMatcher()`.  
A matcher function is defined as `func(*http.Request, *MockRequest) error`

```go
var getUserMock = apitest.NewMock().
    Post("http://example.com/user/12345").
    AddMatcher(func(req *http.Request, mockReq *MockRequest) error {
    	if req.Method == http.MethodPost {
    		return nil
    	}
    	return errors.New("invalid http method")
    }).
    RespondWith().
    Status(http.StatusOK).
    End()
```

## Standalone Mode

You can use mocks outside of API tests by using the `EndStandalone` termination method on the mock builder. This is useful for testing http clients outside of api tests.

```go
func TestMocks_Standalone(t *testing.T) {
	cli := http.Client{Timeout: 5}
	defer NewMock().
		Post("http://localhost:8080/path").
		Body(`{"a", 12345}`).
		RespondWith().
		Status(http.StatusCreated).
		EndStandalone()()

	resp, err := cli.Post("http://localhost:8080/path",
		"application/json",
		strings.NewReader(`{"a", 12345}`))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

`EndStandalone` returns a function that should be invoked after the test runs to reset the http transport to the default configuration.

If you want to register multiple standalone mocks in a test, use the `apitest.NewStandaloneMocks()` factory method.

```go
resetTransport := apitest.NewStandaloneMocks(
	apitest.NewMock().
		Post("http://localhost:8080/path").
		Body(`{"a": 12345}`).
		RespondWith().
		Status(http.StatusCreated).
		End(),
	apitest.NewMock().
		Get("http://localhost:8080/path").
		RespondWith().
		Body(`{"a": 12345}`).
		Status(http.StatusOK).
		End(),
).End()
defer resetTransport()
```

<!-- TODO: explain the matchers -->
