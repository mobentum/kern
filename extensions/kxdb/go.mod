module github.com/mobentum/kern/extensions/kxdb

go 1.26.5

require (
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/mobentum/kern v1.0.1
	github.com/mobentum/xdb v0.2.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mobentum/kern => ../../
