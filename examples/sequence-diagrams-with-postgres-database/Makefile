postgres-dev:
	docker stop apitest_postgres || true && docker rm apitest_postgres || true
	docker run -d -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=apitest -p 5432:5432 --name apitest_postgres postgres:9.6

test:
	POSTGRES_DSN="host=localhost port=5432 user=postgres password=postgres dbname=apitest sslmode=disable" go test ./... -v
