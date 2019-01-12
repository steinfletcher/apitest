package apitest

import (
	"fmt"
	"net/http"
	"time"
)

type expectedCookie struct {
	name     *string
	value    *string
	path     *string
	domain   *string
	expires  *time.Time
	maxAge   *int
	secure   *bool
	httpOnly *bool
}

func ExpectedCookie(name string) *expectedCookie {
	return &expectedCookie{
		name: &name,
	}
}

func (cookie *expectedCookie) Value(value string) *expectedCookie {
	cookie.value = &value
	return cookie
}

func (cookie *expectedCookie) Path(path string) *expectedCookie {
	cookie.path = &path
	return cookie
}

func (cookie *expectedCookie) Domain(domain string) *expectedCookie {
	cookie.domain = &domain
	return cookie
}

func (cookie *expectedCookie) Expires(expires time.Time) *expectedCookie {
	cookie.expires = &expires
	return cookie
}

func (cookie *expectedCookie) MaxAge(maxAge int) *expectedCookie {
	cookie.maxAge = &maxAge
	return cookie
}

func (cookie *expectedCookie) Secure(secure bool) *expectedCookie {
	cookie.secure = &secure
	return cookie
}

func (cookie *expectedCookie) HttpOnly(httpOnly bool) *expectedCookie {
	cookie.httpOnly = &httpOnly
	return cookie
}

func (cookie *expectedCookie) ToHttpCookie() *http.Cookie {
	httpCookie := http.Cookie{}

	if cookie.name != nil {
		httpCookie.Name = *cookie.name
	}

	if cookie.value != nil {
		httpCookie.Value = *cookie.value
	}

	if cookie.path != nil {
		httpCookie.Path = *cookie.path
	}

	if cookie.domain != nil {
		httpCookie.Domain = *cookie.domain
	}

	if cookie.expires != nil {
		httpCookie.Expires = *cookie.expires
	}

	if cookie.maxAge != nil {
		httpCookie.MaxAge = *cookie.maxAge
	}

	if cookie.secure != nil {
		httpCookie.Secure = *cookie.secure
	}

	if cookie.httpOnly != nil {
		httpCookie.HttpOnly = *cookie.httpOnly
	}

	return &httpCookie
}

// Compares cookies based on only the provided fields from expectedCookie.
// Supported fields are Name, Value, Domain, Path, Expires, MaxAge, Secure and HttpOnly
func compareCookies(expectedCookie *expectedCookie, actualCookie *http.Cookie) (bool, []string) {
	cookieFound := *expectedCookie.name == actualCookie.Name
	compareErrors := []string{}

	if cookieFound {

		formatError := func(name string, expectedValue, actualValue interface{}) string {
			return fmt.Sprintf("Missmatched field %s. Expected %v but received %v",
				name,
				expectedValue,
				actualValue)
		}

		if expectedCookie.value != nil && *expectedCookie.value != actualCookie.Value {
			compareErrors = append(compareErrors, formatError("Value", *expectedCookie.value, actualCookie.Value))
		}

		if expectedCookie.domain != nil && *expectedCookie.domain != actualCookie.Domain {
			compareErrors = append(compareErrors, formatError("Domain", *expectedCookie.value, actualCookie.Domain))
		}

		if expectedCookie.path != nil && *expectedCookie.path != actualCookie.Path {
			compareErrors = append(compareErrors, formatError("Path", *expectedCookie.path, actualCookie.Path))
		}

		if expectedCookie.expires != nil && !(*expectedCookie.expires).Equal(actualCookie.Expires) {
			compareErrors = append(compareErrors, formatError("Expires", *expectedCookie.expires, actualCookie.Expires))
		}

		if expectedCookie.maxAge != nil && *expectedCookie.maxAge != actualCookie.MaxAge {
			compareErrors = append(compareErrors, formatError("MaxAge", *expectedCookie.maxAge, actualCookie.MaxAge))
		}

		if expectedCookie.secure != nil && *expectedCookie.secure != actualCookie.Secure {
			compareErrors = append(compareErrors, formatError("Secure", *expectedCookie.secure, actualCookie.Secure))
		}

		if expectedCookie.httpOnly != nil && *expectedCookie.httpOnly != actualCookie.HttpOnly {
			compareErrors = append(compareErrors, formatError("HttpOnly", *expectedCookie.httpOnly, actualCookie.HttpOnly))
		}
	}

	return cookieFound, compareErrors
}
