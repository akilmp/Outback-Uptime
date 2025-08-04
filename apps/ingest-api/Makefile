APP=ingest-api

build:
	go build -o bin/$(APP) ./cmd/server

run: build
	./bin/$(APP)

test:
	go test ./...

docker-build:
	docker build -t $(APP):latest .

.PHONY: build run test docker-build
