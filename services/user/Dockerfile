FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
    -buildvcs=false -trimpath -ldflags="-w -s" \
    -o /app/main ./services/user/cmd

FROM alpine:3.23
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/main ./main
COPY services/user/db/migrations ./db/migrations
ENV CONFIG_PATH=/app/config.yaml

EXPOSE 50051
CMD ["./main"]
