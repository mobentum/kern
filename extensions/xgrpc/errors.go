package xgrpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatusCoder allows domain errors to define grpc status codes.
type StatusCoder interface {
	GRPCStatusCode() codes.Code
}

// ToStatus maps regular errors into grpc status errors.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if coder, ok := err.(StatusCoder); ok {
		return status.Error(coder.GRPCStatusCode(), err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
