FROM golang:1.20

WORKDIR /app

COPY go.mod .
COPY cmd/main.go ./cmd/main.go
COPY Makefile .

RUN make build
COPY build/daily-reporter .

ENTRYPOINT [ "/app/daily-reporter" ]
