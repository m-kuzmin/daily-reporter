.PHONY: default check \
	docker-build docker-run \
	lint test run build

# Frequently used commands

default: lint run

run: build
	./build/daily-reporter

lint:
	golangci-lint run

test:
	go test ./...

check: lint test build/daily-reporter docker-build

# Building the app

build: api/github/generated.go
	mkdir -p build
	CGO_ENABLED=0 GOOS=linux go build -o build/daily-reporter cmd/*.go

# Github GraphQL API

api/github/generated.go: api/github/genqlient.yaml \
	api/github/schema.graphql \
	$(wildcard internal/clients/github/*.go)
	go run github.com/Khan/genqlient api/github/genqlient.yaml

api/github/schema.graphql:
	wget -O api/github/schema.graphql https://docs.github.com/public/schema.docs.graphql

# Docker commands

docker-build:
	docker build . -t daily-reporter:latest --rm

docker-run: docker-build
	docker run daily-reporter:latest

