.PHONY: test test-examples lint docs fmt vet

test:
	bash -c 'diff -u <(echo -n) <(go fmt $(go list ./...))'
	go vet ./...
	go test ./... -v -race -covermode=atomic -coverprofile=coverage.out

test-examples:
	cd examples && go test -v ./... && \
	cd sequence-diagrams-with-sqlite-database && make test && cd ..

lint:
	golangci-lint run ./...

docs:
	cd docs && hugo server -w && cd -
