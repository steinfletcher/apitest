*This repo is experimental and isn't ready to be consumed*

# api-test

[![Build Status](https://travis-ci.org/steinfletcher/api-test.svg?branch=master)](https://travis-ci.org/steinfletcher/api-test) [![Coverage Status](https://coveralls.io/repos/github/steinfletcher/api-test/badge.svg?branch=master)](https://coveralls.io/github/steinfletcher/api-test?branch=master)

Simple behavioural (black box) api testing library. 

_In black box tests the internal structure of the app is not know by the tests. Data is input by calling the rest endpoints with a http client and the outputs are expected to meet certain conditions._

## Installation

     go get -u github.com/steinfletcher/api-test

## Example

```go
func TestGetUser(t *testing.T) {
    apitest.New(handler).
      Get("/user/1234").
      Expect(t).
      Body(`{"id": "1234", "name": "Tate"}`).
      Status(http.StatusCreated).
      End()
}
```
