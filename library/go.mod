module github.com/bakurvik/mylib/library

go 1.23.6

require (
	github.com/bakurvik/mylib/common v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bakurvik/mylib/common => ../common
