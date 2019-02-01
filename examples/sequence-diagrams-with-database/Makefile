
postgres-dev:
	docker stop apitest || true && docker rm apitest || true
	docker run -d -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=apitest -p 5432:5432 --name apitest postgres:9.6
	sleep 4
	cd migrations && goose postgres "host=localhost port=5432 user=postgres password=postgres dbname=apitest sslmode=disable" up
