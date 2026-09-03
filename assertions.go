package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

func (a *APITest) assertMocks() {
	for _, mock := range a.mocks {
		if !mock.anyTimesSet && !mock.isUsed && mock.timesSet {
			a.verifier.Fail(a.t, "mock was not invoked expected times", failureMessageArgs{Name: a.name})
		}
	}
}

func (a *APITest) assertFunc(res *http.Response, req *http.Request) {
	if len(a.response.assert) > 0 {
		for _, assertFn := range a.response.assert {
			err := assertFn(copyHttpResponse(res), copyHttpRequest(req))
			if err != nil {
				a.verifier.NoError(a.t, err, failureMessageArgs{Name: a.name})
			}
		}
	}
}

func (a *APITest) assertResponse(res *http.Response) {
	if a.response.status != 0 {
		a.verifier.Equal(a.t, a.response.status, res.StatusCode, fmt.Sprintf("Status code %d not equal to %d", res.StatusCode, a.response.status), failureMessageArgs{Name: a.name})
	}

	if a.response.body != "" {
		var resBodyBytes []byte
		if res.Body != nil {
			resBodyBytes, _ = io.ReadAll(res.Body)
			res.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
		}
		if json.Valid([]byte(a.response.body)) {
			a.verifier.JSONEq(a.t, a.response.body, string(resBodyBytes), failureMessageArgs{Name: a.name})
		} else {
			a.verifier.Equal(a.t, a.response.body, string(resBodyBytes), failureMessageArgs{Name: a.name})
		}
	}
}

func (a *APITest) assertCookies(response *http.Response) {
	actualCookies := response.Cookies()

	for _, expectedCookie := range a.response.cookies {
		var mismatchedFields []string
		foundCookie := false
		for _, actualCookie := range actualCookies {
			cookieFound, errors := compareCookies(expectedCookie, actualCookie)
			if cookieFound {
				foundCookie = true
				mismatchedFields = append(mismatchedFields, errors...)
			}
		}
		a.verifier.Equal(a.t, true, foundCookie, "ExpectedCookie not found - "+*expectedCookie.name, failureMessageArgs{Name: a.name})
		a.verifier.Equal(a.t, 0, len(mismatchedFields), strings.Join(mismatchedFields, ","), failureMessageArgs{Name: a.name})
	}

	for _, cookieName := range a.response.cookiesPresent {
		a.verifier.Equal(a.t, true, hasCookie(actualCookies, cookieName), "ExpectedCookie not found - "+cookieName, failureMessageArgs{Name: a.name})
	}

	for _, cookieName := range a.response.cookiesNotPresent {
		a.verifier.Equal(a.t, false, hasCookie(actualCookies, cookieName), "ExpectedCookie found - "+cookieName, failureMessageArgs{Name: a.name})
	}
}

func hasCookie(cookies []*http.Cookie, name string) bool {
	return slices.ContainsFunc(cookies, func(cookie *http.Cookie) bool {
		return cookie.Name == name
	})
}

func (a *APITest) assertHeaders(res *http.Response) {
	for expectedHeader, expectedValues := range a.response.headers {
		resHeaderValues, foundHeader := res.Header[expectedHeader]
		a.verifier.Equal(a.t, true, foundHeader, fmt.Sprintf("expected header '%s' not present in response", expectedHeader), failureMessageArgs{Name: a.name})

		if foundHeader {
			for _, expectedValue := range expectedValues {
				foundValue := slices.Contains(resHeaderValues, expectedValue)
				a.verifier.Equal(a.t, true, foundValue, fmt.Sprintf("mismatched values for header '%s'. Expected %s but received %s", expectedHeader, expectedValue, strings.Join(resHeaderValues, ",")), failureMessageArgs{Name: a.name})
			}
		}
	}

	if len(a.response.headersPresent) > 0 {
		for _, expectedName := range a.response.headersPresent {
			if res.Header.Get(expectedName) == "" {
				a.verifier.Fail(a.t, fmt.Sprintf("expected header '%s' not present in response", expectedName), failureMessageArgs{Name: a.name})
			}
		}
	}

	if len(a.response.headersNotPresent) > 0 {
		for _, name := range a.response.headersNotPresent {
			if res.Header.Get(name) != "" {
				a.verifier.Fail(a.t, fmt.Sprintf("did not expect header '%s' in response", name), failureMessageArgs{Name: a.name})
			}
		}
	}
}
