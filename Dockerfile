# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o vitalbridge .

# Runtime stage
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/vitalbridge /app/vitalbridge
COPY config/config.yaml /app/config/

RUN adduser -D -s /bin/sh appuser

WORKDIR /app
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["./vitalbridge"]
