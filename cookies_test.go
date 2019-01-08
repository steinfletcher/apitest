package apitest

import (
	"net/http"
	"testing"
	"time"
)

func TestApiTest_Cookies_ExpectedCookie(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	cookie := NewCookie("Tom").
		Value("LovesBeers").
		Path("/at-the-lyric").
		Domain("in.london").
		Expires(expiry).
		MaxAge(10).
		Secure(true).
		HttpOnly(false)

	ten := 10
	boolt := true
	boolf := false

	assertEqual(t, Cookie{
		name:     toString("Tom"),
		value:    toString("LovesBeers"),
		path:     toString("/at-the-lyric"),
		domain:   toString("in.london"),
		expires:  &expiry,
		maxAge:   &ten,
		secure:   &boolt,
		httpOnly: &boolf,
	}, *cookie)
}

func TestApiTest_Cookies_ToHttpCookie(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	httpCookie := NewCookie("Tom").
		Value("LovesBeers").
		Path("/at-the-lyric").
		Domain("in.london").
		Expires(expiry).
		MaxAge(10).
		Secure(true).
		HttpOnly(false).
		ToHttpCookie()

	assertEqual(t, http.Cookie{
		Name:     "Tom",
		Value:    "LovesBeers",
		Path:     "/at-the-lyric",
		Domain:   "in.london",
		Expires:  expiry,
		MaxAge:   10,
		Secure:   true,
		HttpOnly: false,
	}, *httpCookie)
}

func TestApiTest_Cookies_ToHttpCookie_PartiallyCreated(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	httpCookie := NewCookie("Tom").
		Value("LovesBeers").
		Expires(expiry).
		ToHttpCookie()

	assertEqual(t, http.Cookie{
		Name:     "Tom",
		Value:    "LovesBeers",
		Expires:  expiry,
		Secure:   false,
		HttpOnly: false,
	}, *httpCookie)
}

func toString(str string) *string {
	return &str
}
