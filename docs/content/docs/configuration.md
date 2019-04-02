# Configuration

The `APITest` configuration type exposes some methods to register test hooks, enabled debug logging and to define the handler under test.

## Handler

Define the handler that should be tested as follows, where `myHandler` is a `http.Handler`

```go
apitest.New().
	Handler(myHandler)
```

`apitest` does not make a http call over the network. Instead the provided http handler's `ServeHTTP` method is invoked in the same process as the test code. The user defined request and response structs are converted to `http.Request` and `http.Response` types via Go's `httptest` package. The goal here is to test the internal application and not the networking layer. This approach keeps the test fast simple. If you would like to use a real http client to generate a request against a running application we recommend using a tool like [Baloo](https://github.com/h2non/baloo) which has a similar api to but the principles are much different.

## Debug Output

Enabling debug logging will write the http wire representation of all request and response interactions to the console.

```go
apitest.New().
	Debug().
	Handler(myHandler)
```

This will also log mock interactions. This can be useful for identifying the root cause behind test failures related to unmatched mocks. In this example the mocks do not match due to an incorrect URL in the mock definition. The request is compared with each registered mock and the reason is logged to the console for each mock mismatch

```text
----------> inbound http request
GET /user HTTP/1.1
Host: application

failed to match mocks. Errors: received request did not match any mocks

Mock 1 mismatches:
• received path /user/12345 did not match mock path /preferences/12345

Mock 2 mismatches:
• received path /user/12345 did not match mock path /user/123456

----------> request to mock
GET /user/12345 HTTP/1.1
Host: localhost:8080
User-Agent: Go-http-client/1.1
Accept-Encoding: gzip
...

```

## Hooks

### Observe

### Intercept

### Reporting
