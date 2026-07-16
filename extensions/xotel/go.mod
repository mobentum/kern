module github.com/mobentum/kern/extensions/xotel

go 1.25.12

require (
	github.com/mobentum/kern v0.1.2
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/trace v1.35.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
)

replace github.com/mobentum/kern => ../../

