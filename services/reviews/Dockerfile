FROM golang:1.26
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN go build -o reviews-service ./services/reviews/cmd
CMD ["./reviews-service", "-listen", "0.0.0.0:50051"]
