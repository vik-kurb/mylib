#!/bin/sh
set -e

set -a
[ -f "/.env" ] && . /.env
set +a

until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
  echo "Waiting for DB..."
  sleep 1
done

goose -dir /schema postgres "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

echo "All env vars:"
env

exec ./users
