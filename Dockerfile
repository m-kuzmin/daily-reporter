FROM golang:1.20

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY api api
COPY cmd cmd
COPY internal internal

COPY Makefile ./
RUN make build

# // TODO: Use multistage builds so that final image is lighter
COPY build/daily-reporter .
COPY assets assets
COPY config.toml ./

ENTRYPOINT [ "/app/daily-reporter" ]
