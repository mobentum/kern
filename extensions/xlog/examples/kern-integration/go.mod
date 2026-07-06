module github.com/mobentum/kern/extensions/xlog/examples/kern-integration

go 1.25.0

require (
	github.com/mobentum/kern v0.0.0
	github.com/mobentum/kern/extensions/xlog v0.0.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace github.com/mobentum/kern => ../../../../

replace github.com/mobentum/kern/extensions/xlog => ../../
