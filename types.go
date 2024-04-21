package main

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
)

// Function is the general runtime of the composition function
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer
	log logging.Logger
}
