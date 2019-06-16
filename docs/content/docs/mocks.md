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
