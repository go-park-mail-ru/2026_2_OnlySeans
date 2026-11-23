
FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG BUILD_PATH=./api-gateway/cmd

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false -trimpath -ldflags="-w -s" \
    -o /app/main "$BUILD_PATH"

FROM alpine:3.23
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/main ./main

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 8080 50051 50052
CMD ["./main"]
