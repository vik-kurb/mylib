FROM golang:1.23.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o library

FROM alpine:latest

COPY --from=builder /app/library /library
COPY --from=builder /app/.env /.env

WORKDIR /

EXPOSE 8080

CMD ["./library"]