# m-kuzmin/daily-repoter Makefile
#
# Users should use meta actions, while meta actions utilize atomic
# operations (like build, run) that have no dependencies or side
# effects. This allows the user to compose a meta action without
# an atomic action like `build` or `run` triggering `lint` twice
.PHONY: default check all docker build run docker-build docker-run lint test

# Meta actions

# Not very pedantic run
default: lint build run
# Checks everything
check: lint build test docker-build
# Builds all targets
all: docker-build build
# First time docker prep
docker: docker-build docker-run

# Native/Local (i.e. not docker)

build:
	mkdir -p build
	go build -o build/daily-reporter cmd/main.go
run:
	go run cmd/main.go

# Docker

docker-build:
	docker build . -t daily-reporter:latest
docker-run:
	docker run daily-reporter:latest

# Helpers/Additional commands

lint:
	golangci-lint run
test:
	go test ./...

