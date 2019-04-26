.PHONY: test test-examples docs fmt vet

test:
	go test ./... -v -race -covermode=atomic -coverprofile=coverage.out

test-examples:
	cd examples && go test -v -race ./... && cd ..

fmt:
	bash -c 'diff -u <(echo -n) <(gofmt -s -d ./)'

vet:
	bash -c 'diff -u <(echo -n) <(go vet ./...)'

test-all: fmt vet test test-examples

docs:
	cd docs && hugo server -w && cd -
