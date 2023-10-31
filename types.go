package main

import (
	"encoding/json"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/giantswarm/xfnlib/pkg/composite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer
	log       logging.Logger
	composed  *composite.Composition
	composite XRObject
}

// XRSpec is the definition of the XR as an object
type XRSpec struct {
	Labels            map[string]string `json:"labels"`
	ProviderConfigRef string            `json:"providerConfigRef"`
	DeletionPolicy    string            `json:"deletionPolicy"`
	ClaimRef          struct {
		Namespace string `json:"namespace"`
	} `json:"claimRef"`

	CompositionSelector struct {
		MatchLabels struct {
			Provider string `json:"provider"`
		} `json:"matchLabels"`
	} `json:"compositionSelector"`
}

type XRObject struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     XRSpec            `json:"spec"`
}

// VpcConfig Describes what the VPC looks like
type VpcConfig struct {
	VpcID   string   `json:"vpcId"`
	Subnets []string `json:"subnetIds"`
}

// ClusterAtProvider Contains information about what real infrastructure exists for this type
type ClusterAtProvider struct {
	VpcConfig []VpcConfig `json:"vpcConfig,omitempty"`
}

// ClusterStatus is the status of the composed resource that we require
type ClusterStatus struct {
	AtProvider *ClusterAtProvider `json:"atProvider,omitempty"`
}

// ClusterForProvider describes what information is given to the cluster provider
type ClusterForProvider struct {
	Region string `json:"region"`
}

// Policy Policies for referencing.
type Policy struct {
	Resolution string `json:"resolution,omitempty"`
	Resolve    string `json:"resolve,omitempty"`
}

// ProviderConfigRef specifies how the provider that will be used to create, observe, update, and delete this managed resource should be configured.
type ProviderConfigRef struct {
	Name   string `json:"name"`
	Policy Policy `json:"policy,omitempty"`
}

// ConnectionSecretRef specifies the namespace and name of a Secret to which any connection details for this managed resource should be written.
type ConnectionSecretRef struct {
	Namespace string `json:"namespace"`
}

// ClusterSpec describes the spec given to the cluster
type ClusterSpec struct {
	ForProvider      ClusterForProvider  `json:"forProvider"`
	ProviderConfig   ProviderConfigRef   `json:"providerConfigRef"`
	ConnectionSecret ConnectionSecretRef `json:"writeConnectionSecretToRef"`
}

// Metadata information to pass between objects
type Metadata struct {
	Annotations map[string]string      `json:"annotations"`
	Labels      map[string]interface{} `json:"labels"`
}

// ClusterObject is the information we are going to pull from unstructured
type ClusterObject struct {
	Metadata Metadata      `json:"metadata"`
	Spec     ClusterSpec   `json:"spec"`
	Status   ClusterStatus `json:"status"`
}

type SubnetObject struct {
	Status SubnetStatus `json:"status"`
}

type SubnetStatus struct {
	AtProvider *Subnet `json:"atProvider,omitempty"`
}

func (s *SubnetStatus) UnmarshalJSON(data []byte) (err error) {
	var (
		inf map[string]interface{}
		sn  Subnet
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

type Subnet struct {
	ID                  string            `json:"id"`
	AvailabilityZone    string            `json:"availabilityZone"`
	CidrBlock           string            `json:"cidrBlock"`
	IsIpv6              bool              `json:"isIpV6"`
	Ipv6CidrBlock       string            `json:"ipv6CidrBlock"`
	Tags                map[string]string `json:"tags"`
	IsPublic            bool              `json:"isPublic"`
	MapPublicIPOnLaunch *bool             `json:"mapPublicIpOnLaunch,omitempty"`
}
