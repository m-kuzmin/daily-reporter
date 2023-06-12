FROM golang:1.20

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY config.toml Makefile ./
COPY cmd cmd
COPY internal internal
COPY api api

RUN make build
COPY build/daily-reporter .

ENTRYPOINT [ "/app/daily-reporter" ]
