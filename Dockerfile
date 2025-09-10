FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o vaultwarden-syncer ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/vaultwarden-syncer .
COPY config.yaml.example config.yaml

RUN mkdir -p /app/data

EXPOSE 8181

VOLUME ["/app/data", "/vaultwarden-data"]

CMD ["./vaultwarden-syncer"]