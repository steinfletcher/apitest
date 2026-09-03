<p align="center">
    <img src="https://apitest.dev/static/images/dummy.svg" width="150">
</p>

<p align="center">
<a href="https://pkg.go.dev/github.com/steinfletcher/apitest"><img src="https://pkg.go.dev/badge/github.com/steinfletcher/apitest.svg" alt="Go Reference" /></a>
<a href="https://github.com/steinfletcher/apitest/actions/workflows/ci.yml"><img src="https://github.com/steinfletcher/apitest/actions/workflows/ci.yml/badge.svg" alt="Build Status" /></a>
<a href="https://goreportcard.com/report/github.com/steinfletcher/apitest"><img src="https://goreportcard.com/badge/github.com/steinfletcher/apitest" alt="Go Report Card" /></a>
<a href="https://github.com/avelino/awesome-go/#testing"><img src="https://awesome.re/mentioned-badge.svg" alt="Mentioned in Awesome Go" /></a>
</p>

# apitest

A simple and extensible behavioural testing library for Go HTTP services. Tests are written the way a client uses the
API: build a request, send it to the handler, and assert on the response. External HTTP calls can be mocked, and every
test can render a sequence diagram of what happened.

In behavioural tests the internal structure of the app is not known by the tests. Data is input to the system and the
outputs are expected to meet certain conditions.

- Fluent builders for requests and expectations
- Works with anything that implements `http.Handler`, or against a running server
- Declarative mocks for outbound HTTP calls, with matchers for every part of the request
- Sequence diagrams of each test, including mocked calls and database queries
- No third party dependencies; integrates with `testing`, Ginkgo and custom assertion libraries

The API is stable. The library is maintained and issues are addressed; feature requests are considered.

Join the conversation at #apitest on [https://gophers.slack.com](https://gophers.slack.com).

<span>Logo by <a target="_blank" href="https://twitter.com/egonelbre">@egonelbre</a><span>

## Documentation

This README covers the whole library. The same material with more narrative is at
[https://apitest.dev](https://apitest.dev), and the API reference is on
[pkg.go.dev](https://pkg.go.dev/github.com/steinfletcher/apitest).

## Installation

```bash
go get github.com/steinfletcher/apitest
```

## Quick start

```go
func TestGetUser(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/user/1234").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"id": "1234", "name": "Tate"}`).
		End()
}
```

`Handler` takes any `http.Handler`, so a `http.ServeMux`, a gin engine, an echo instance and so on all work. Everything
before `Expect(t)` describes the request; everything after it describes the expected response. `End()` runs the test.
When the expected body is valid JSON it is compared structurally, so key order and whitespace do not matter.

## Demo

![animated gif](./apitest.gif)

## Building the request

### Method and URL

`Get`, `Post`, `Put`, `Patch` and `Delete` set the method and URL. Each has an `f` variant that formats the URL, and
`Method` covers anything else.

```go
apitest.Handler(handler).Getf("/user/%s", id)

apitest.Handler(handler).Method(http.MethodOptions).URL("/user")
```

A ready-made `*http.Request` can be used instead of the builder.

```go
req := httptest.NewRequest(http.MethodGet, "/user/1234", nil)

apitest.Handler(handler).
	HttpRequest(req).
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Headers

```go
apitest.Handler(handler).
	Get("/hello").
	Header("Authorization", "Bearer token").
	Headers(map[string]string{"X-Request-Id": "12345"}).
	ContentType("application/json").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Query parameters

`Query`, `QueryParams` and `QueryCollection` can be combined. `QueryCollection` sets repeated parameters, so
`map[string][]string{"a": {"b", "c", "d"}}` is encoded as `a=b&a=c&a=d`.

```go
apitest.Handler(handler).
	Get("/hello").
	QueryParams(map[string]string{"a": "1", "b": "2"}).
	Query("c", "d").
	QueryCollection(map[string][]string{"e": {"f", "g"}}).
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Cookies

```go
apitest.Handler(handler).
	Get("/hello").
	Cookie("session", "12345").
	Cookies(apitest.NewCookie("theme").Value("dark").Path("/")).
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Body

`Body` sets a raw body. `JSON` sets the body and the `Content-Type` header; it accepts a string, a `[]byte`, or any
value that can be marshalled. `BodyFromFile` and `JSONFromFile` read the body from disk.

```go
apitest.Handler(handler).
	Post("/user").
	JSON(map[string]any{"name": "jan", "age": 32}).
	Expect(t).
	Status(http.StatusCreated).
	End()
```

GraphQL requests are built with `GraphQLQuery`, or `GraphQLRequest` when an operation name is needed.

```go
apitest.Handler(handler).
	Post("/graphql").
	GraphQLQuery(`query User($id: ID!) { user(id: $id) { name } }`, map[string]any{"id": "1234"}).
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Form data

`FormData` sends an `application/x-www-form-urlencoded` body.

```go
apitest.Handler(handler).
	Post("/hello").
	FormData("a", "1").
	FormData("b", "2", "3").
	Expect(t).
	Status(http.StatusOK).
	End()
```

`MultipartFormData` and `MultipartFile` send `multipart/form-data`. The two form styles cannot be combined in one
request.

```go
apitest.Handler(handler).
	Post("/upload").
	MultipartFormData("description", "holiday photos").
	MultipartFile("file", "testdata/beach.jpg", "testdata/sunset.jpg").
	Expect(t).
	Status(http.StatusOK).
	End()
```

Files are read from the OS by default. `UseFS` swaps in any `fs.FS`, such as an in-memory `fstest.MapFS`.

```go
inMemFS := fstest.MapFS{
	"audio.wav": &fstest.MapFile{Data: []byte{19, 2, 123, 12, 35, 1}},
}

apitest.Handler(handler).
	UseFS(inMemFS).
	Post("/upload").
	MultipartFile("file", "audio.wav").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Basic auth

```go
apitest.Handler(handler).
	Get("/hello").
	BasicAuth("username", "password").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Context

`WithContext` sets the request context, which is how deadlines, cancellation and context values reach the handler.

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

apitest.Handler(handler).
	Get("/hello").
	WithContext(ctx).
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Intercept

`Intercept` receives the built `*http.Request` just before it is sent, for changes the builder does not cover.

```go
apitest.Handler(handler).
	Intercept(func(req *http.Request) {
		req.URL.RawQuery = "a[]=xxx&a[]=yyy"
	}).
	Get("/hello").
	Expect(t).
	Status(http.StatusOK).
	End()
```

## Asserting on the response

### Status and body

```go
apitest.Handler(handler).
	Get("/user/1234").
	Expect(t).
	Status(http.StatusOK).
	Body(`{"id": "1234", "name": "Tate"}`).
	End()
```

A JSON body is compared structurally. Any other body is compared as an exact string. `Bodyf` formats the expected
body and `BodyFromFile` reads it from disk.

### Headers

```go
apitest.Handler(handler).
	Get("/hello").
	Expect(t).
	Status(http.StatusOK).
	Header("Content-Type", "application/json").
	Headers(map[string]string{"X-Request-Id": "12345"}).
	HeaderPresent("Etag").
	HeaderNotPresent("X-Powered-By").
	End()
```

### Cookies

Only the fields set on the expected cookie are compared, so `NewCookie("session").Value("12345")` ignores the path,
expiry and other attributes of the actual cookie.

```go
apitest.Handler(handler).
	Patch("/hello").
	Expect(t).
	Status(http.StatusOK).
	Cookie("session", "12345").
	Cookies(apitest.NewCookie("theme").Value("dark").HttpOnly(true)).
	CookiePresent("csrf").
	CookieNotPresent("legacy").
	End()
```

### Custom assertions

`Assert` takes a function that receives copies of the response and request and returns an error on failure. It can
be called several times.

```go
apitest.Handler(handler).
	Get("/hello").
	Expect(t).
	Assert(func(res *http.Response, req *http.Request) error {
		if res.Header.Get("X-Rate-Limit") == "" {
			return errors.New("expected a rate limit header")
		}
		return nil
	}).
	End()
```

`apitest.IsSuccess`, `apitest.IsClientError` and `apitest.IsServerError` are ready-made assertions on the status code
range.

### JSONPath

For asserting on parts of the response body, the separate
[apitest-jsonpath](https://github.com/steinfletcher/apitest-jsonpath) module provides JSONPath assertions. It is
packaged separately to keep this library dependency free.

Given the response `{"a": 12345, "b": [{"key": "c", "value": "result"}]}`:

```go
apitest.Handler(handler).
	Get("/hello").
	Expect(t).
	Assert(jsonpath.Contains(`$.b[? @.key=="c"].value`, "result")).
	Assert(jsonpath.Equal(`$.a`, float64(12345))).
	End()
```

### The result

`End()` returns a `Result` holding the response, so further checks can be made after the test has run.

```go
result := apitest.Handler(handler).
	Get("/user/1234").
	Expect(t).
	Status(http.StatusOK).
	End()

var user struct {
	Name string `json:"name"`
}
result.JSON(&user)
```

## Mocking external HTTP calls

If the handler under test calls other services, those calls can be answered by mocks. A mock describes the request to
match and the response to return.

```go
var getUser = apitest.NewMock().
	Get("http://users/api/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()

var getPreferences = apitest.NewMock().
	Get("http://preferences/api/preferences/12345").
	Header("Authorization", "Bearer .*").
	RespondWith().
	JSON(map[string]any{"is_contactable": true}).
	Status(http.StatusOK).
	End()

func TestGetUser(t *testing.T) {
	apitest.New().
		Mocks(getUser, getPreferences).
		Handler(handler).
		Get("/user/12345").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "is_contactable": true}`).
		End()
}
```

The request side supports `Header`, `Headers`, `Query`, `QueryParams`, `QueryCollection`, `Body`, `Bodyf`,
`BodyFromFile`, `BodyRegexp`, `JSON`, `FormData`, `Cookie` and `BasicAuth`, plus `Present` and `NotPresent`
variants for headers, query parameters, form data and cookies. The response side supports `Status`, `Body`, `Bodyf`,
`BodyFromFile`, `JSON`, `Header`, `Headers`, `Cookie` and `Cookies`. When no `Content-Type` is set on the response,
`application/json` is used for a JSON body and `text/plain` otherwise.

Mocks work by replacing the transport of `http.DefaultClient` for the duration of the test. If the code under test
uses its own `http.Client`, pass it with `HttpClient` so its transport is replaced instead. Because the default
transport is process-wide, tests that use mocks should not run in parallel unless each provides its own client.

```go
apitest.New().
	HttpClient(client).
	Mocks(getUser).
	Handler(handler).
	Get("/user/12345").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### How mocks are matched

Mocks are tried in the order they are passed to `Mocks`, and the first mock whose matchers all pass is used. Each mock
is used once unless `Times` or `AnyTimes` is set. The built-in matchers behave as follows:

* **Path** must equal the mock path, or match it as a regular expression. The expression is not anchored, so
  `Get("/user")` also matches `/user/1234`; use `Get("^/user$")` for an exact match.
* **Host** and **scheme** must equal the mock's when the mock URL specifies them, otherwise they are ignored.
* **Method** must equal the mock's.
* **Header**, **query** and **form data** values are regular expressions matched against the received values.
  `HeaderPresent`, `QueryPresent`, `FormDataPresent` and `CookiePresent` (and their `NotPresent` counterparts) check
  for presence regardless of value.
* **Body** must equal the mock body, or be equivalent JSON. `BodyRegexp` matches the body against a regular expression.
* **Cookies** and **basic auth** must equal the mock's.

When no mock matches, the call fails with an error listing why each mock was rejected. Run the test with `Debug()` to
see it.

### Custom matchers

`AddMatcher` adds a matcher alongside the built-in ones. It receives the actual request and the mock's request
specification, and returns an error when the request should not match.

```go
var getPreferences = apitest.NewMock().
	Get("/preferences/12345").
	AddMatcher(func(r *http.Request, mr *apitest.MockRequest) error {
		if r.URL.Scheme != "https" {
			return errors.New("expected an https request")
		}
		return nil
	}).
	RespondWith().
	Status(http.StatusOK).
	End()
```

### Times and unused mocks

By default a mock answers one request. `Times(n)` makes it answer `n` requests and fails the test if it is called
fewer times. `AnyTimes` lets a mock answer any number of calls, including none, and takes precedence over `Times`.

```go
var getUser = apitest.NewMock().
	Get("http://users/api/user/12345").
	RespondWith().
	Status(http.StatusOK).
	Times(2).
	End()

var healthCheck = apitest.NewMock().
	Get("http://users/health").
	RespondWith().
	Status(http.StatusOK).
	AnyTimes().
	End()
```

Mocks that were never called are reported on the result.

```go
result := apitest.New().
	Mocks(getUser, getPreferences).
	Handler(handler).
	Get("/user/12345").
	Expect(t).
	Status(http.StatusOK).
	End()

for _, unmatched := range result.UnmatchedMocks() {
	t.Logf("mock not called: %s", unmatched.URL.String())
}
```

### Slow and failing dependencies

`Timeout` makes the mocked call fail with a timeout error. `FixedDelay` waits the given number of milliseconds before
responding, and only takes effect when the test enables delays with `EnableMockResponseDelay`.

```go
var slowUsers = apitest.NewMock().
	Get("http://users/api/user/12345").
	RespondWith().
	FixedDelay(500).
	Status(http.StatusOK).
	End()

var brokenPreferences = apitest.NewMock().
	Get("http://preferences/api/preferences/12345").
	RespondWith().
	Timeout().
	End()

func TestGetUser_WhenDependenciesAreSlow(t *testing.T) {
	apitest.New().
		EnableMockResponseDelay().
		Mocks(slowUsers, brokenPreferences).
		Handler(handler).
		Get("/user/12345").
		Expect(t).
		Status(http.StatusGatewayTimeout).
		End()
}
```

### Observing mock calls

`ObserveMocks` is called with the request and response of every mocked call.

```go
apitest.New().
	ObserveMocks(func(res *http.Response, req *http.Request, apiTest *apitest.APITest) {
		t.Logf("%s %s -> %d", req.Method, req.URL, res.StatusCode)
	}).
	Mocks(getUser).
	Handler(handler).
	Get("/user/12345").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Standalone mocks

Mocks can also be used outside an apitest test, for example in a unit test of an HTTP client. `EndStandalone` and
`NewStandaloneMocks(...).End()` install the mocks and return a function that removes them.

```go
func TestUserClient(t *testing.T) {
	reset := apitest.NewMock().
		Get("http://users/api/user/12345").
		RespondWith().
		Body(`{"name": "jon"}`).
		Status(http.StatusOK).
		EndStandalone()
	defer reset()

	user, err := userClient.Get("12345")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "jon" {
		t.Fatalf("unexpected user %+v", user)
	}
}
```

Use `HttpClient` on the mock to target a specific client, and `Debug` on the mock to log the matching.

## Sequence diagrams and reports

`Report` renders every test into a report once it completes. The built-in `SequenceDiagram` formatter writes an HTML
page per test with a sequence diagram of the request, any mocked calls, and the response, along with the full wire
representation of each message. See the
[demo](http://demo-html.apitest.dev.s3-website-eu-west-1.amazonaws.com/).

```go
func TestGetUser(t *testing.T) {
	apitest.New("gets the user").
		Report(apitest.SequenceDiagram()).
		Mocks(getUser, getPreferences).
		Handler(handler).
		Get("/user/12345").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

The name passed to `New` appears in the report. Diagrams are written to `.sequence` by default; pass a path to
`SequenceDiagram` to change that. The participants are labelled `cli` and `sut` unless `Meta` provides names.

```go
apitest.New("gets the user").
	Report(apitest.SequenceDiagram(".sequence-diagrams")).
	Meta(map[string]any{
		"consumerName":        "web-app",
		"systemUnderTestName": "user-api",
	}).
	Handler(handler).
	Get("/user/12345").
	Expect(t).
	Status(http.StatusOK).
	End()
```

### Database calls in diagrams

The `x/db` package wraps a `database/sql` driver so that queries made during the test appear in the diagram. Wrap the
driver with a `Recorder`, register the wrapped driver, open the database through it, and pass the same recorder to the
test.

```go
var recorder = apitest.NewTestRecorder()

func init() {
	sql.Register("recorded-sqlite3", apitestdb.WrapWithRecorder("sqlite3", recorder))
}

func TestGetUser(t *testing.T) {
	db, err := sql.Open("recorded-sqlite3", "./users.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	apitest.New("gets the user").
		Recorder(recorder).
		Report(apitest.SequenceDiagram()).
		Handler(newApp(db)).
		Get("/user/12345").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

`WrapConnectorWithRecorder` does the same for a `driver.Connector`. See the
[sqlite](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-sqlite-database),
[mysql](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-mysql-database) and
[postgres](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-postgres-database)
examples.

### Custom reports

Anything implementing `ReportFormatter` can be passed to `Report`. It receives the `Recorder`, which holds the title,
the meta data and the ordered list of events, so reports can be rendered in any format. The
[PlantUML](https://github.com/steinfletcher/apitest-plantuml) companion library is one example. Custom events can be
added to the recorder from your own code with `AddMessageRequest` and `AddMessageResponse`.

## Testing a running server

`EnableNetworking` sends the request over the network instead of to a handler. Pass a client to control timeouts,
cookies and redirects.

```go
client := &http.Client{Timeout: 5 * time.Second}

apitest.New().
	EnableNetworking(client).
	Get("http://localhost:8080/health").
	Expect(t).
	Status(http.StatusOK).
	End()
```

## Debugging

`Debug` prints the wire representation of the inbound request, the final response, and every mocked call, together
with the reasons a request failed to match any mock.

```go
apitest.New().
	Debug().
	Handler(handler).
	Get("/hello").
	Expect(t).
	Status(http.StatusOK).
	End()
```

`Observe` gives programmatic access to the same data.

```go
apitest.New().
	Observe(func(res *http.Response, req *http.Request, apiTest *apitest.APITest) {
		// inspect the copies of res and req
	}).
	Handler(handler).
	Get("/hello").
	Expect(t).
	Status(http.StatusOK).
	End()
```

## Testing handler timeouts

Wrap the handler in the standard library's `http.TimeoutHandler` when passing it to apitest. A handler that overruns
the timeout produces the `503` status and body that `TimeoutHandler` generates, which can be asserted on as usual. To
check that a handler honours a deadline instead, pass a context with a deadline using `WithContext`.

```go
func TestSlowEndpoint(t *testing.T) {
	handler := http.TimeoutHandler(slowHandler, 50*time.Millisecond, "request timed out")

	apitest.Handler(handler).
		Get("/slow").
		Expect(t).
		Status(http.StatusServiceUnavailable).
		Body("request timed out").
		End()
}
```

## Test framework integration

`Expect` accepts any value implementing `TestingT`, which is the subset of `*testing.T` that apitest needs:
`Errorf`, `Fatal` and `Fatalf`. Ginkgo's `GinkgoT()` satisfies it, so apitest works inside Ginkgo specs; see the
[Ginkgo example](https://github.com/steinfletcher/apitest/tree/master/examples/ginkgo).

Assertions are performed through the `Verifier` interface. `Verifier` swaps in a different implementation, for example
`NoopVerifier` to run a test without failing it, or an adapter over your preferred assertion library.

```go
apitest.New().
	Verifier(apitest.NoopVerifier{}).
	Handler(handler).
	Get("/hello").
	Expect(t).
	Status(http.StatusTeapot).
	End()
```

## Examples

### Framework and library integration examples

| Example                                                                                              | Comment                                                                                                    |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| [gin](https://github.com/steinfletcher/apitest/tree/master/examples/gin)                             | popular martini-like web framework                                                                         |
| [graphql](https://github.com/steinfletcher/apitest/tree/master/examples/graphql)                     | using gqlgen.com to generate a graphql server                                                              |
| [gorilla](https://github.com/steinfletcher/apitest/tree/master/examples/gorilla)                     | the gorilla web toolkit                                                                                    |
| [iris](https://github.com/steinfletcher/apitest/tree/master/examples/iris)                           | iris web framework                                                                                         |
| [echo](https://github.com/steinfletcher/apitest/tree/master/examples/echo)                           | High performance, extensible, minimalist Go web framework                                                  |
| [fiber](https://github.com/steinfletcher/apitest/tree/master/examples/fiber)                         | Express inspired web framework written in Go                                                               |
| [httprouter](https://github.com/steinfletcher/apitest/tree/master/examples/httprouter)               | High performance HTTP request router that scales well                                                      |
| [Ginkgo](https://github.com/steinfletcher/apitest/tree/master/examples/ginkgo)                       | Ginkgo BDD test framework                                                                                  |
| [mocks](https://github.com/steinfletcher/apitest/tree/master/examples/mocks)                         | mocking out external http calls, including a custom matcher                                                |
| [websockets](https://github.com/steinfletcher/apitest/tree/master/examples/websockets)               | a websocket handler under test                                                                             |
| [sequence diagrams](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams) | generate sequence diagrams from tests. See the [demo](http://demo-html.apitest.dev.s3-website-eu-west-1.amazonaws.com/) |
| [sqlite](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-sqlite-database), [mysql](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-mysql-database), [postgres](https://github.com/steinfletcher/apitest/tree/master/examples/sequence-diagrams-with-postgres-database) | database queries recorded in sequence diagrams |

### Companion libraries

| Library                                                                 | Comment                                        |
| ----------------------------------------------------------------------- | -----------------------------------------------|
| [JSONPath](https://github.com/steinfletcher/apitest-jsonpath)           | JSONPath assertion addons                      |
| [CSS Selectors](https://github.com/steinfletcher/apitest-css-selector)  | CSS selector assertion addons                  |
| [PlantUML](https://github.com/steinfletcher/apitest-plantuml)           | Export sequence diagrams as plantUML           |
| [DynamoDB](https://github.com/steinfletcher/apitest-dynamodb)           | Add DynamoDB interactions to sequence diagrams |

### Credits

This library was influenced by the following software packages:

* [YatSpec](https://github.com/bodar/yatspec) for creating sequence diagrams from tests
* [MockMVC](https://spring.io) and [superagent](https://github.com/visionmedia/superagent) for the concept and behavioural testing approach
* [Gock](https://github.com/h2non/gock) for the approach to mocking HTTP services in Go
* [Baloo](https://github.com/h2non/baloo) for API design

## Contributing

View the [contributing guide](CONTRIBUTING.md).
