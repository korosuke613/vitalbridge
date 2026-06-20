# Build stage
FROM golang:1.25-alpine@sha256:523c3effe300580ed375e43f43b1c9b091b68e935a7c3a92bfcc4e7ed55b18c2 AS builder

ARG VERSION=dev

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o vitalbridge .

# Runtime stage
FROM alpine:3.22@sha256:310c62b5e7ca5b08167e4384c68db0fd2905dd9c7493756d356e893909057601

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/vitalbridge /app/vitalbridge
COPY config/config.yaml /app/config/

RUN adduser -D -s /bin/sh appuser

WORKDIR /app
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["./vitalbridge"]
