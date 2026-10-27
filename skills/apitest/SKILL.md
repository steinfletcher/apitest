---
name: apitest
description: Write, fix and review Go tests that use github.com/steinfletcher/apitest to test HTTP handlers and APIs, including requests, response assertions, JSONPath assertions, mocks for outbound HTTP calls, sequence diagram reports and debugging failed tests. Use when a Go project imports github.com/steinfletcher/apitest, or when asked to test a Go http.Handler, router or REST API.
license: MIT
---

# apitest

apitest tests a Go HTTP handler the way its clients use it. A test builds a request, sends it to the handler in process, and asserts on the response. Outbound HTTP calls the handler makes are answered by mocks.

The full documentation is at https://apitest.dev. Add `.md` to any docs URL for its Markdown, for example https://apitest.dev/docs/mocks/matching.md. The whole site is at https://apitest.dev/llms-full.txt. Every method is listed in [reference.md](reference.md).

## Setup

```bash
go get github.com/steinfletcher/apitest
go get github.com/steinfletcher/apitest-jsonpath # only for JSONPath assertions
```

apitest needs Go 1.23 or later. Check the version in `go.mod`, because the behaviour marked v1.7.0 below differs in older releases.

## The shape of every test

```go
func TestGetUser(t *testing.T) {
	apitest.New("gets a user").
		Handler(newRouter()).
		Get("/user/1234").
		Header("Authorization", "Bearer token").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"id": "1234", "name": "jon"}`).
		End()
}
```

- Configuration comes first: `Handler`, `Mocks`, `Debug`, `Report` and so on. The name passed to `New` is optional and appears in reports.
- A method such as `Get`, `Post` or `Method` starts the request. Everything up to `Expect(t)` builds it.
- Everything after `Expect(t)` is an expectation. `End()` sends the request and runs the checks.
- `apitest.Handler(h)` is shorthand for `apitest.New().Handler(h)`, and `apitest.HandlerFunc(f)` takes a handler function.

## Rules that prevent most mistakes

1. **End every test with `End()`.** Without it nothing runs and the test passes.
2. **Only what you set is checked.** Without `Status`, the status code is not asserted, so set it in every test.
3. **`Body` compares JSON structurally** when the expected body is valid JSON, ignoring key order and whitespace. Any other body must match exactly, including a trailing newline. For partial matches use JSONPath or `Assert`.
4. **Every failed check is reported** with `t.Errorf`, so read all of the failures, not just the first.
5. **Test the handler in process.** `Handler` calls `ServeHTTP` directly, which is fast and needs no ports. Use `EnableNetworking(client)` with absolute URLs only for end-to-end tests against a running server.
6. **Requests are sent with `Host: sut`.** Set `Host("api.example.com")` when routing depends on the host.
7. **Pass routers, not handler functions, to test routing.** chi, gin, echo, gorilla and `http.ServeMux` routers are all `http.Handler`. Fiber is not, and needs an adapter.

## Building requests

| Method | Use |
| --- | --- |
| `Get`, `Post`, `Put`, `Patch`, `Delete`, and `Getf` and friends | Method and URL. The `f` variants format the URL. |
| `Method(m)` with `URL(u)` | Any other method. |
| `Query(k, v)`, `QueryParams(map)`, `QueryCollection(map[string][]string)` | Query parameters, encoded with sorted keys. |
| `Header(k, v)`, `Headers(map)`, `ContentType(v)` | Headers. `Header` adds a value; `ContentType` replaces. |
| `Cookie(name, value)`, `Cookies(apitest.NewCookie(name).Value(v))` | Cookies. |
| `JSON(v)`, `JSONFromFile(path)` | JSON body plus `Content-Type: application/json`. `v` can be a string, `[]byte` or any marshallable value. |
| `Body(s)`, `Bodyf(format, args...)`, `BodyFromFile(path)` | Raw body. Set the content type yourself. |
| `FormData(k, v...)` | URL encoded form. Cannot be combined with the multipart builders. |
| `MultipartFormData(k, v...)`, `MultipartFile(field, paths...)` | Multipart form. `UseFS(fs.FS)` reads files from memory. |
| `GraphQLQuery(query, vars)`, `GraphQLRequest(body)` | GraphQL POST bodies. |
| `BasicAuth(user, pass)`, `WithContext(ctx)`, `Intercept(func(*http.Request))` | Auth, request context, and changes the builder cannot express. |
| `HttpRequest(req)` | A ready-made `*http.Request`, used as is. |

Since v1.7.0, a query string written into the URL is kept, and builder parameters are appended after it. Earlier versions silently drop it, so on those versions add every parameter with `Query`.

## Asserting on responses

| Method | Checks |
| --- | --- |
| `Status(code)` | The status code. |
| `Body(s)`, `Bodyf(...)`, `BodyFromFile(path)` | The whole body. |
| `Header(k, v)`, `Headers(map)` | That one of the header's values equals `v` exactly. |
| `HeaderPresent(k)`, `HeaderNotPresent(k)` | Header presence. |
| `Cookie(name, value)`, `Cookies(apitest.NewCookie(name).Path("/").HttpOnly(true))` | Cookies. Only the fields you set are compared. |
| `CookiePresent(name)`, `CookieNotPresent(name)` | Cookie presence. |
| `Assert(func(*http.Response, *http.Request) error)` | Anything else. Return an error to fail. `apitest.IsSuccess`, `IsClientError` and `IsServerError` check status ranges. |

`End()` returns a `Result`. `result.Response` is a copy of the response, `result.JSON(&v)` decodes the body once, and `result.UnmatchedMocks()` lists mocks that were never called.

### JSONPath

JSONPath assertions are in a separate module:

```go
func TestGetArticle(t *testing.T) {
	apitest.Handler(newRouter()).
		Get("/articles/12345").
		Expect(t).
		Status(http.StatusOK).
		Assert(jsonpath.Equal(`$.author.name`, "jon")).
		Assert(jsonpath.Equal(`$.id`, float64(12345))).
		Assert(jsonpath.Contains(`$.tags`, "go")).
		Assert(jsonpath.Len(`$.comments`, 2)).
		Assert(jsonpath.NotPresent(`$.author.password`)).
		End()
}
```

Import it as `jsonpath "github.com/steinfletcher/apitest-jsonpath"`. It has three traps:

- **Numbers decode as `float64`.** Compare with `float64(12345)`, not `12345`.
- **`Present` means non-empty.** A field that is `false`, `0`, `""`, `null` or `[]` counts as not present. Check those values with `Equal`.
- **`GreaterThan(expr, n)` and `LessThan(expr, n)` are inclusive.** They mean a length of at least `n` and at most `n`.

Filters select from arrays, as in `$.comments[?@.approved == true].author`. `jsonpath.Chain()` and `jsonpath.Root("$.author")` group several assertions.

## Mocking outbound HTTP calls

```go
var getUserMock = apitest.NewMock().
	Get("http://users/api/user/1234").
	RespondWith().
	Status(http.StatusOK).
	JSON(map[string]any{"id": "1234", "name": "jon"}).
	End()

func TestGetProfile(t *testing.T) {
	apitest.New().
		Mocks(getUserMock).
		Handler(newRouter()).
		Get("/profile/1234").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

How mocks behave:

- **Mocks replace `http.DefaultTransport` for the test.** `http.Get`, `http.DefaultClient` and any `http.Client` without its own `Transport` are mocked with no code changes. A client with its own `Transport` bypasses the mocks. Make that client injectable, give the same client to the code under test, and pass it with `HttpClient(client)`.
- **There is no fall-through to the network.** A call that matches no mock fails with an error, and the handler usually responds with an error status.
- **Always set `Status`.** A mock response defaults to status `0`.
- **Paths and values are unanchored regular expressions.** `Get("http://users/api/user")` also matches `/api/user/1234`, and `Query("page", "1")` matches `page=10`. Anchor with `$`, or leave out the scheme and host to anchor both ends, as in `Get("^/api/user/1234$")`. Escape `.`, `?` and `+`.
- **Mocks are tried in order, and each answers one call.** `Times(n)` answers `n` calls and fails the test if fewer are made. `AnyTimes()` answers any number. An unused mock without `Times` does not fail the test.
- **`Mocks(...)` replaces earlier mocks**, so pass them all in one call.
- **Mocked tests must not run in parallel** with `t.Parallel()` while they share `http.DefaultTransport`. Give each parallel test its own `*http.Client` and pass it with `HttpClient`.

Request matchers go between the method and `RespondWith()`:

- `Header`, `Headers`, `HeaderPresent` and `HeaderNotPresent`.
- `Query`, `QueryParams`, `QueryCollection`, `QueryPresent` and `QueryNotPresent`.
- `Body` and `JSON`, which match exactly or as equivalent JSON, and `BodyRegexp`.
- `FormData`, `FormDataPresent` and `FormDataNotPresent`.
- `Cookie`, `CookiePresent`, `CookieNotPresent` and `BasicAuth`.
- `AddMatcher(func(*http.Request, *apitest.MockRequest) error)` for anything else. The `mocks` package of `apitest-jsonpath` provides JSONPath matchers, as in `AddMatcher(mocks.Equal("$.name", "jon"))`.

Methods without a helper use `Method(http.MethodOptions).URL("http://users/api/user")`. Since v1.7.0, a `Method` mock without a URL matches any URL. On earlier versions it panics.

The response side sets `Status`, `Header`, `Headers`, `Cookie`, `Cookies`, and the body with `Body`, `Bodyf`, `JSON` or `BodyFromFile`. Without a `Content-Type`, a JSON body gets `application/json` and anything else gets `text/plain`.

To test failure paths, respond with an error status, or use `Timeout()`, which makes the call fail with a network timeout error. `FixedDelay(ms)` only applies when the test calls `EnableMockResponseDelay()`, and it is a plain sleep that does not trigger client timeouts.

### Standalone mocks

To unit test an HTTP client without an apitest test, install mocks with `EndStandalone`, which returns a function that removes them:

```go
func TestUsersClient_GetUser(t *testing.T) {
	defer apitest.NewMock().
		Get("http://users/api/user/1234").
		RespondWith().
		Status(http.StatusOK).
		Body(`{"id": "1234", "name": "jon"}`).
		EndStandalone()()

	user, err := NewUsersClient("http://users").GetUser("1234")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "jon" {
		t.Fatalf("unexpected user %+v", user)
	}
}
```

`apitest.NewStandaloneMocks(m1, m2).End()` installs several mocks at once. Since v1.7.0 standalone mocks honour `Times`, but standalone mode never verifies call counts.

## Debugging a failed test

Add `Debug()` to the configuration and run `go test -run TestName -v`. It prints the request into the handler, every mocked call, and the final response. When a call matches no mock, it lists why each mock was rejected:

```text
Mock 1 mismatches:
• received path /api/user/1234 did not match mock path /api/user/12345
```

The usual causes are:

- A path, host or query value that differs from the mock.
- A mock that already answered its one call.
- A client with its own `Transport` that bypasses the mocks.
- A missing `Status` on the mock.

Remove `Debug()` once the test passes.

## Patterns

Cover several scenarios with a table:

```go
func TestGetProfile_Errors(t *testing.T) {
	tests := []struct {
		name       string
		mock       *apitest.Mock
		wantStatus int
	}{
		{
			name: "user not found",
			mock: apitest.NewMock().
				Get("http://users/api/user/1234").
				RespondWith().
				Status(http.StatusNotFound).
				End(),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "users service down",
			mock: apitest.NewMock().
				Get("http://users/api/user/1234").
				RespondWith().
				Status(http.StatusServiceUnavailable).
				End(),
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apitest.New(tt.name).
				Mocks(tt.mock).
				Handler(newRouter()).
				Get("/profile/1234").
				Expect(t).
				Status(tt.wantStatus).
				End()
		})
	}
}
```

Share configuration with a helper, because every configuration method returns `*apitest.APITest`:

```go
func newAPITest(name string) *apitest.APITest {
	return apitest.New(name).
		Report(apitest.SequenceDiagram()).
		Handler(newRouter())
}

func TestGetProfile_Reported(t *testing.T) {
	newAPITest("gets a profile").
		Mocks(getUserMock).
		Get("/profile/1234").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

More patterns:

- **Sequence diagrams.** `Report(apitest.SequenceDiagram())` writes an HTML report per test to `.sequence/`, which should be in `.gitignore`. Name the participants with `Meta(map[string]any{"consumerName": "web", "systemUnderTestName": "user-api"})`.
- **Database queries in diagrams.** `github.com/steinfletcher/apitest/x/db` wraps a `database/sql` driver with `WrapWithRecorder(driverName, recorder)`. Pass the same recorder to the test with `Recorder(recorder)`.
- **Handler timeouts.** Wrap the handler in `http.TimeoutHandler` and expect `503`.
- **Ginkgo.** Pass `GinkgoT()` to `Expect`.
- **Capturing without failing.** `Verifier(apitest.NoopVerifier{})` runs a test without asserting.
