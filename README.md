*This repo is experimental and isn't ready to be consumed*

# api-test

Simple behavioural (black box) api testing library. 

_In black box tests the internal structure of the app is not know by the tests. Data is input by calling the rest endpoints with a http client and the outputs are expected to meet certain conditions._

## Example

```go
func TestGetCustomer(t *testing.T) {
    New(handler).
      Get("/hello").
      Expect(t).
      Body(`{"a": 12345}`).
      Status(http.StatusCreated).
      End()
}
```
