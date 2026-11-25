FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false -trimpath -ldflags="-w -s" \
    -o /app/main ./services/persons/cmd

FROM alpine:3.23
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/main ./main
COPY services/persons/db/migrations ./db/migrations
ENV CONFIG_PATH=/app/config.yaml

EXPOSE 50053
CMD ["./main"]
