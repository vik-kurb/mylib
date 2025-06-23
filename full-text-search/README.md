## full-text-search
Microservice that indexes and searches data using Elasticsearch.

Environment variables should be set in .env:
| Variable      | Description                              | Example                                                            |
| ------------- | ---------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`     | Name of the main application database    | `full_text_search`                                                          |
| `DB_HOST`     | Hostname of the PostgreSQL server        | `db` (Docker service name)                                         |
| `DB_PORT`     | Port on which PostgreSQL is listening    | `5432`                                                             |
| `DB_USER`     | Database user                            | `postgres`                                                         |
| `DB_PASSWORD` | Database user password                   | `postgres`                                                         |
| `TEST_DB_URL` | Connection URL for test database (local) | `postgres://postgres:@localhost:5432/test_full_text_search?sslmode=disable` |
