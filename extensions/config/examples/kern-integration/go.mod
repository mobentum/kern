module github.com/mobentum/kern/extensions/config/examples/kern-integration

go 1.25.0

require (
	github.com/mobentum/kern v0.0.0
	github.com/mobentum/kern/extensions/config v0.0.0
)

require github.com/joho/godotenv v1.5.1 // indirect

replace github.com/mobentum/kern => ../../../../

replace github.com/mobentum/kern/extensions/config => ../../
