package main

import (
	"encoding/json"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fnc "github.com/giantswarm/crossplane-fn-generate-subnets/pkg/composite/v1beta1"
)

// Function is the general runtime of the composition function
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer
	log logging.Logger
}

// VpcConfig Describes what the VPC looks like
type VpcConfig struct {
	VpcID   string   `json:"vpcId"`
	Subnets []string `json:"subnetIds"`
}

// ClusterAtProvider Contains information about what real infrastructure exists
// for this type
type ClusterAtProvider struct {
	VpcConfig []VpcConfig `json:"vpcConfig,omitempty"`
}

// ProviderConfigRef specifies how the provider that will be used to
// create, observe, update, and delete this managed resource should be
// configured
type ProviderConfigRef struct {
	Name   string `json:"name"`
	Policy struct {
		Resolution string `json:"resolution,omitempty"`
		Resolve    string `json:"resolve,omitempty"`
	} `json:"policy,omitempty"`
}

// ClusterSpec describes the spec given to the cluster
type ClusterSpec struct {
	ForProvider struct {
		Region string `json:"region"`
	} `json:"forProvider"`
	ProviderConfig   ProviderConfigRef `json:"providerConfigRef"`
	ConnectionSecret struct {
		Namespace string `json:"namespace"`
	} `json:"writeConnectionSecretToRef"`
}

// ClusterObject is the information we are going to pull from unstructured
type ClusterObject struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     ClusterSpec       `json:"spec"`
	Status   struct {
		AtProvider *ClusterAtProvider `json:"atProvider,omitempty"`
	} `json:"status"`
}

// AWSSubnetObject is a wrapper for the information required from the provider
// subnet
//
// Most of the subnet information coming from the provider is discarded by this
// object as the only element of intererst is the status
type AwsSubnetObject struct {
	Status AwsSubnetStatus `json:"status"`
}

// AWSSubnetStatus holds the information required by the function thats stored
// in the provider resource object
//
// Most of the information from the status is discarded as the only relevant
// information is stored in `status.atProvider`
type AwsSubnetStatus struct {
	AtProvider *fnc.AwsSubnet `json:"atProvider,omitempty"`
}

// UnmarshalJSON is a custom unmarshaller for the AWSSubnetStatus object
//
// This function prevents the subnet from being unmarshalled if its ID field is
// empty.
func (s *AwsSubnetStatus) UnmarshalJSON(data []byte) (err error) {
	var (
		inf map[string]interface{}
		sn  fnc.AwsSubnet
	)
	s.AtProvider = nil

	if err = json.Unmarshal(data, &inf); err != nil {
		return
	}

	if _, ok := inf["atProvider"]; !ok {
		return
	}

	str, _ := json.Marshal(inf["atProvider"])
	if err = json.Unmarshal(str, &sn); err != nil {
		return
	}
	if sn.ID != "" {
		s.AtProvider = &sn
	}
	return
}
