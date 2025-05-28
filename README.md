# mylib

Backend system for managing a digital library using Go microservices. System consists of 3 microservices: library, users and user-reading.


## Running with Docker Compose

```bash
docker-compose up --build
```

- http://localhost:8080 → library API

- http://localhost:8081 → users API

- http://localhost:8082 → user-reading API

- http://localhost:8080/swagger/index.html → Swagger UI for library

- http://localhost:8081/swagger/index.html → Swagger UI for users

- http://localhost:8082/swagger/index.html → Swagger UI for user-reading


## library
Microservice that stores books and authors data. [API](./library/README.md)

Environment variables should be set in .env:
| Variable                   | Description                               | Example                                                            |
| -------------------------- | ----------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`                  | Name of the main application database     | `library`                                                          |
| `DB_HOST`                  | Hostname of the PostgreSQL server         | `db` (Docker service name)                                         |
| `DB_PORT`                  | Port on which PostgreSQL is listening     | `5432`                                                             |
| `DB_USER`                  | Database user                             | `postgres`                                                         |
| `DB_PASSWORD`              | Database user password                    | `postgres`                                                         |
| `TEST_DB_URL`              | Connection URL for test database (local)  | `postgres://postgres:@localhost:5432/test_library?sslmode=disable` |
| `MAX_SEARCH_BOOKS_LIMIT`   | Maximum number of books found in search   | `10`                                                               |
| `MAX_SEARCH_AUTHORS_LIMIT` | Maximum number of authors found in search | `10`                                                               |


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

## user-reading
Microservice that stores reading status of user books. [API](./user-reading/README.md)

Environment variables should be set in .env:
| Variable      | Description                              | Example                                                            |
| ------------- | ---------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`     | Name of the main application database    | `user_reading`                                                          |
| `DB_HOST`     | Hostname of the PostgreSQL server        | `db` (Docker service name)                                         |
| `DB_PORT`     | Port on which PostgreSQL is listening    | `5432`                                                             |
| `DB_USER`     | Database user                            | `postgres`                                                         |
| `DB_PASSWORD` | Database user password                   | `postgres`                                                         |
| `TEST_DB_URL` | Connection URL for test database (local) | `postgres://postgres:@localhost:5432/test_user_reading?sslmode=disable` |
| `USERS_SERVICE_HOST` | Host of users service | `http://users:8080` |
| `LIBRARY_SERVICE_HOST` | Host of library service | `http://library:8080` |


## License

This project is licensed under the MIT License – see the [LICENSE](./LICENSE) file for details.