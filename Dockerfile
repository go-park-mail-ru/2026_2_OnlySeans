FROM golang:1.26 AS source
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM source AS onlineconf-build
RUN CGO_ENABLED=0 go build -o onlineconf ./cmd/onlineconf

FROM alpine:3.23 AS onlineconf
WORKDIR /app
COPY --from=onlineconf-build /app/onlineconf .
COPY configs/ ./configs/
CMD ["./onlineconf"]

FROM source AS backend
RUN go build -o app ./cmd
CMD ["./app"]
