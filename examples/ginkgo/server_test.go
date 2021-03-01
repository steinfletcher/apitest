package server_test

import (
	"net/http"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	server "github.com/steinfletcher/apitest/examples/ginkgo"
)

var _ = Describe("Ginkgo/Server", func() {

	var (
		t      GinkgoTInterface
		router *mux.Router
	)

	BeforeEach(func() {
		t = GinkgoT()
		router = server.NewApp().Router
	})

	Context("Successful CookieMatching", func() {
		It("cookies should be set correctly", func() {
			apitest.New().
				Handler(router).
				Get("/user/1234").
				Expect(t).
				Cookies(apitest.NewCookie("TomsFavouriteDrink").
					Value("Beer").
					Path("/")).
				Status(http.StatusOK).
				End()
		})
	})

	Context("Successful GetUser", func() {
		It("Get User body should return desired value", func() {
			apitest.New().
				Handler(router).
				Get("/user/1234").
				Expect(t).
				Body(`{"id": "1234", "name": "Andy"}`).
				Status(http.StatusOK).
				End()
		})

		It("Get User jsonpath should return desired value", func() {
			apitest.New().
				Handler(router).
				Get("/user/1234").
				Expect(t).
				Assert(jsonpath.Equal(`$.id`, "1234")).
				Status(http.StatusOK).
				End()
		})
	})

	Context("Unsuccessful GetUser", func() {
		It("User not found error should be raised", func() {
			apitest.New().
				Handler(router).
				Get("/user/1515").
				Expect(t).
				Status(http.StatusNotFound).
				End()
		})
	})
})
