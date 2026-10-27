# apitest API reference

Every exported identifier in `github.com/steinfletcher/apitest` v1.7, grouped by where it is used. Signatures and doc comments are on https://pkg.go.dev/github.com/steinfletcher/apitest.

## Starting a test

| Function | Returns | Notes |
| --- | --- | --- |
| `New(name ...string)` | `*APITest` | The optional name appears in reports. |
| `Handler(http.Handler)` | `*APITest` | Shorthand for `New().Handler(h)`. |
| `HandlerFunc(http.HandlerFunc)` | `*APITest` | Shorthand for `New().HandlerFunc(f)`. |

## APITest

Configuration methods return `*APITest`. Request methods return `*Request`.

| Method | Notes |
| --- | --- |
| `Handler(http.Handler)`, `HandlerFunc(http.HandlerFunc)` | The handler under test. |
| `Mocks(...*Mock)` | Mocks for outbound calls. Replaces earlier mocks. |
| `HttpClient(*http.Client)` | Mock this client's transport instead of `http.DefaultTransport`. |
| `EnableMockResponseDelay()` | Honour `FixedDelay` on mocks. |
| `ObserveMocks(Observe)` | Called for every mocked call. The response is nil when no mock matched. |
| `Observe(...Observe)` | Called with the final response. Replaces earlier observers. |
| `Debug()` | Print every request and response, and mock mismatch reasons. |
| `Report(ReportFormatter)` | Render the test, for example `Report(SequenceDiagram())`. |
| `Recorder(*Recorder)` | Use this recorder, for custom events and `x/db`. |
| `Meta(map[string]any)` | Values for reports. `consumerName` and `systemUnderTestName` rename the `cli` and `sut` participants. |
| `Intercept(func(*http.Request))` | Change the built request before it is sent. One per test. |
| `EnableNetworking(...*http.Client)` | Send the request over the network. Needs absolute URLs. |
| `Verifier(Verifier)` | Replace the assertion implementation, for example `NoopVerifier{}`. |
| `UseFS(fs.FS)` | File system for `MultipartFile`. |
| `Host(string)` | The request `Host`. Defaults to `sut`. |
| `Get`, `Post`, `Put`, `Patch`, `Delete` `(url)` | Start the request. |
| `Getf`, `Postf`, `Putf`, `Patchf`, `Deletef` `(format, args...)` | Start the request with a formatted URL. |
| `Method(string)` | Start the request with any method. Set the URL with `Request.URL`. |
| `HttpRequest(*http.Request)` | Use a ready-made request as is. Builder methods do not change it. |

## Request

Methods return `*Request`. `Expect(t)` returns `*Response`.

| Method | Notes |
| --- | --- |
| `URL(string)`, `URLf(format, args...)` | The URL. A query string in it is kept since v1.7.0. |
| `Header(k, v)`, `Headers(map[string]string)` | Add header values. Names are canonicalised. |
| `ContentType(string)` | Set `Content-Type`, replacing earlier values. |
| `Query(k, v)`, `QueryParams(map[string]string)` | Add query parameters. |
| `QueryCollection(map[string][]string)` | Repeated parameters. A second call replaces the first. |
| `Cookie(name, value)`, `Cookies(...*Cookie)` | Add cookies. |
| `Body(string)`, `Bodyf(format, args...)`, `BodyFromFile(path)` | Raw body. |
| `JSON(any)`, `JSONFromFile(path)` | JSON body and `Content-Type: application/json`. |
| `FormData(name, values...)` | URL encoded form body. |
| `MultipartFormData(name, values...)`, `MultipartFile(name, paths...)` | Multipart body. Cannot be combined with `FormData`. |
| `GraphQLQuery(query, variables...)`, `GraphQLRequest(GraphQLRequestBody)` | GraphQL POST body. |
| `BasicAuth(user, pass)` | Basic auth header. |
| `WithContext(context.Context)` | Request context, for deadlines and context values. |
| `Expect(TestingT)` | End the request. Takes `*testing.T`, `GinkgoT()` or anything with `Errorf`, `Fatal` and `Fatalf`. |

## Response

Methods return `*Response`. `End()` returns `Result`.

| Method | Notes |
| --- | --- |
| `Status(int)` | Status code. Not checked unless set. |
| `Body(string)`, `Bodyf(format, args...)`, `BodyFromFile(path)` | Whole body. JSON is compared structurally, anything else exactly. |
| `Header(k, v)`, `Headers(map[string]string)` | One of the header's values equals `v`. |
| `HeaderPresent(k)`, `HeaderNotPresent(k)` | Header presence, whatever the value. |
| `Cookie(name, value)`, `Cookies(...*Cookie)` | Cookies. Only the fields set on the expected cookie are compared. |
| `CookiePresent(name)`, `CookieNotPresent(name)` | Cookie presence. |
| `Assert(func(*http.Response, *http.Request) error)` | Custom assertion. Runs after the built-in checks. |
| `End()` | Run the test. |

## Result

| Member | Notes |
| --- | --- |
| `Response *http.Response` | Copy of the final response with a readable body. |
| `JSON(any)` | Decode the body. Reads it, so call once. Panics on invalid JSON. |
| `UnmatchedMocks() []UnmatchedMock` | Mocks no request matched. Each has a `URL url.URL`. |

## Mock

`NewMock()` returns `*Mock`. Verb methods return `*MockRequest`.

| Method | Notes |
| --- | --- |
| `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head` `(url)` | Match the method and URL. The path is an unanchored regular expression. Host and scheme must be equal when given. |
| `Getf`, `Postf`, `Putf`, `Patchf`, `Deletef` `(format, args...)` | As above with a formatted URL. |
| `Method(string)` | Match any method. Matches any URL unless `MockRequest.URL` sets one (v1.7.0). |
| `HttpClient(*http.Client)`, `Debug()` | Options for standalone mocks. |
| `Matches(*http.Request) []error` | Run the matchers against a request. |

## MockRequest

Matchers return `*MockRequest`. `RespondWith()` returns `*MockResponse`. Header, query and form values are unanchored regular expressions.

| Method | Matches |
| --- | --- |
| `URL(url)`, `URLf(format, args...)` | The URL, for a mock started with `Method`. Added in v1.7.0. |
| `Header(k, pattern)`, `Headers(map[string]string)` | Header values. |
| `HeaderPresent(k)`, `HeaderNotPresent(k)` | Header presence. |
| `Query(k, pattern)`, `QueryParams(map[string]string)`, `QueryCollection(map[string][]string)` | Query values. |
| `QueryPresent(k)`, `QueryNotPresent(k)` | Query presence. |
| `Body(string)`, `Bodyf(format, args...)`, `BodyFromFile(path)`, `JSON(any)` | The body, exactly or as equivalent JSON. |
| `BodyRegexp(pattern)` | The body as a regular expression. |
| `FormData(k, patterns...)`, `FormDataPresent(k)`, `FormDataNotPresent(k)` | URL encoded form fields. |
| `Cookie(name, value)`, `CookiePresent(name)`, `CookieNotPresent(name)` | Cookies. |
| `BasicAuth(user, pass)` | Basic auth credentials. |
| `AddMatcher(Matcher)` | Custom matcher, `func(*http.Request, *MockRequest) error`. A matcher that reads the body must restore it. |
| `RespondWith()` | Start the response. |

## MockResponse

Methods return `*MockResponse`. `End()` returns `*Mock`.

| Method | Notes |
| --- | --- |
| `Status(int)` | Status code. Defaults to `0`, so always set it. |
| `Header(k, v)`, `Headers(map[string]string)` | Response headers. |
| `Cookie(name, value)`, `Cookies(...*Cookie)` | `Set-Cookie` headers. |
| `Body(string)`, `Bodyf(format, args...)`, `BodyFromFile(path)`, `JSON(any)` | The body. Without a `Content-Type`, JSON gets `application/json` and other bodies `text/plain`. |
| `Times(int)` | Answer `n` calls. Fails the test if fewer are made, except in standalone mode. |
| `AnyTimes()` | Answer any number of calls. Takes precedence over `Times`. |
| `Timeout()` | Fail the call with a network timeout error. |
| `FixedDelay(ms int64)` | Sleep before responding. Needs `EnableMockResponseDelay()` and ignores client timeouts. |
| `End()` | Finish the mock. |
| `EndStandalone(...*Mock) func()` | Install outside a test. Call the returned function to remove the mocks. |

## Standalone mocks

| Function or method | Notes |
| --- | --- |
| `NewStandaloneMocks(...*Mock)` | Group mocks. |
| `HttpClient(*http.Client)`, `Debug()` | Options. |
| `End() func()` | Install and return the reset function. |

## Cookies

`NewCookie(name)` returns a builder with `Value`, `Path`, `Domain`, `Expires`, `MaxAge`, `Secure` and `HttpOnly`. `FromHTTPCookie(*http.Cookie)` and `ToHttpCookie()` convert to and from `*http.Cookie`.

## Reports

| Name | Notes |
| --- | --- |
| `SequenceDiagram(path ...string)` | HTML sequence diagram formatter. Writes to `.sequence` by default. |
| `ReportFormatter` | Interface with `Format(*Recorder)` for custom reports. |
| `NewTestRecorder()`, `Recorder` | `Title`, `SubTitle`, `Meta` and ordered `Events`. |
| `Recorder.AddMessageRequest`, `AddMessageResponse` | Add custom events, such as queue messages, to reports. |
| `HttpRequest`, `HttpResponse`, `MessageRequest`, `MessageResponse` | Event types. |

The `Meta` passed to formatters also holds `name`, `method`, `path`, `status_code`, `duration` in nanoseconds, and `hash`.

## Extension types

| Type | Signature |
| --- | --- |
| `Assert` | `func(*http.Response, *http.Request) error` |
| `Matcher` | `func(*http.Request, *MockRequest) error` |
| `Observe` | `func(*http.Response, *http.Request, *APITest)` |
| `Intercept` | `func(*http.Request)` |
| `TestingT` | `Errorf`, `Fatal` and `Fatalf` |
| `Verifier` | `Equal`, `True`, `JSONEq`, `Fail` and `NoError`. Implemented by `DefaultVerifier` and `NoopVerifier`. |

`IsSuccess`, `IsClientError` and `IsServerError` are ready-made `Assert` values for the 200 to 399, 400 to 499 and 500 and above status ranges.

## Sub-packages and companion modules

| Package | Contents |
| --- | --- |
| `github.com/steinfletcher/apitest/x/db` | `WrapWithRecorder(driverName, recorder)` and `WrapConnectorWithRecorder(connector, name, recorder)` record SQL queries in reports. |
| `github.com/steinfletcher/apitest/mocks` | `MockVerifier`, a stubbable `Verifier`. |
| `github.com/steinfletcher/apitest-jsonpath` | `Equal`, `NotEqual`, `Contains`, `Present`, `NotPresent`, `Matches`, `Len`, `GreaterThan`, `LessThan`, `JWTHeaderEqual`, `JWTPayloadEqual`, `Chain` and `Root`. |
| `github.com/steinfletcher/apitest-jsonpath/mocks` | JSONPath matchers for mock request bodies: `Equal`, `NotEqual`, `Contains`, `Len` and `GreaterThan`. |
| `github.com/steinfletcher/apitest-css-selector` | CSS selector assertions for HTML responses. |
