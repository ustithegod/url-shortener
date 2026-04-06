FROM golang:1.25.3-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY config ./config
COPY internal ./internal
COPY migrations ./migrations

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/url-shortener ./cmd/url-shortener
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrator ./cmd/migrator

FROM alpine:3.21 AS app

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/url-shortener /usr/local/bin/url-shortener
COPY config ./config

EXPOSE 8082

ENTRYPOINT ["/usr/local/bin/url-shortener"]
CMD ["--config=/app/config/local.yaml"]

FROM alpine:3.21 AS migrator

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/migrator /usr/local/bin/migrator
COPY migrations ./migrations

ENTRYPOINT ["/usr/local/bin/migrator"]
CMD ["--migrations-path=/app/migrations"]
