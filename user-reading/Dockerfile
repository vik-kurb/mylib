FROM golang:1.24.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN git clone https://github.com/pressly/goose.git /goose-src && \
    cd /goose-src/cmd/goose && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/goose

RUN CGO_ENABLED=0 GOOS=linux go build -o user-reading

FROM alpine:latest

RUN apk add --no-cache postgresql-client

WORKDIR /

COPY --from=builder /app/user-reading /user-reading
COPY --from=builder /app/.env /.env
COPY --from=builder /app/goose /usr/local/bin/goose
COPY --from=builder /app/entrypoint.sh /entrypoint.sh
COPY --from=builder /app/sql/schema /schema

RUN chmod +x /entrypoint.sh /usr/local/bin/goose

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]