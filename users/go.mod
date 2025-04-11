module github.com/bakurvik/mylib/users

go 1.23.6

require (
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	github.com/bakurvik/mylib/common v0.0.0
)

replace github.com/bakurvik/mylib/common => ../common

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pingcap/log v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
