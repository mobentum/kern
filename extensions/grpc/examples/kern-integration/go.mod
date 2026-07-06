module github.com/mobentum/kern/extensions/grpc/examples/kern-integration

go 1.25.0

require (
	github.com/mobentum/kern v0.0.0
	github.com/mobentum/kern/extensions/grpc v0.0.0
)

require (
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/mobentum/kern => ../../../../

replace github.com/mobentum/kern/extensions/grpc => ../../
