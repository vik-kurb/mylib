# mylib

Backend system for managing a digital library using Go microservices. System consists of 2 microservices: library and users.


## Running with Docker Compose

```bash
docker-compose up --build
```

- http://localhost:8080 → library API

- http://localhost:8081 → users API

- http://localhost:8080/swagger/index.html → Swagger UI for library

- http://localhost:8081/swagger/index.html → Swagger UI for users


## library
Microservice that stores books and authors data. [API](./library/README.md)

Environment variables should be set in .env:
| Variable      | Description                              | Example                                                            |
| ------------- | ---------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`     | Name of the main application database    | `library`                                                          |
| `DB_HOST`     | Hostname of the PostgreSQL server        | `db` (Docker service name)                                         |
| `DB_PORT`     | Port on which PostgreSQL is listening    | `5432`                                                             |
| `DB_USER`     | Database user                            | `postgres`                                                         |
| `DB_PASSWORD` | Database user password                   | `postgres`                                                         |
| `TEST_DB_URL` | Connection URL for test database (local) | `postgres://postgres:@localhost:5432/test_library?sslmode=disable` |


## users
Microservice that stores users data. [API](./users/README.md)

Environment variables should be set in .env:
| Variable          | Description                                             | Example                                                            |
| ----------------- | ------------------------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`         | Name of the main application database                   | `library`                                                          |
| `DB_HOST`         | Hostname of the PostgreSQL server                       | `db` (Docker service name)                                         |
| `DB_PORT`         | Port on which PostgreSQL is listening                   | `5432`                                                             |
| `DB_USER`         | Database user                                           | `postgres`                                                         |
| `DB_PASSWORD`     | Database user password                                  | `postgres`                                                         |
| `TEST_DB_URL`     | Connection URL for test database (local)                | `postgres://postgres:@localhost:5432/test_library?sslmode=disable` |
| `AUTH_SECRET_KEY` | Secret key used for signing and verifying JWT tokens    | `Q4uTGasVKJUqlpvhlpQ/Lkg3i+3z5LLdkUPH2tjO1dEVWUqnb9VGjPBhV2rAXh63` |

## License

This project is licensed under the MIT License – see the [LICENSE](./LICENSE) file for details.