// Package v1beta1 contains the definition of the XR requirements for using this function
// +groupName=definition
// +versionName=v1beta1
package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane;composition;functions;subnets

// XrObjectDefinition contains information about the XR
//
// This type is a meta-type for defining the XRD spec as it excludes
// fields normally defined as part of a standard XRD definition
type XrObjectDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec The specification of the XR
	Spec XrClaimSpec `json:"spec"`

	// Status information about the status of the XR
	Status XRStatus `json:"status"`
}

type CompositeObject struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec The specification of the XR
	Spec XrSpec `json:"spec"`

	// Status information about the status of the XR
	Status XRStatus `json:"status"`
}

type XrClaimSpec struct {
	// Labels is a set of additional labels to be applied to all objects
	// +optional
	// +mapType=granular
	Labels map[string]string `json:"labels"`

	// Defines the name of the providerconfig for the cloud provider
	// +kubebuilder:validation:Required
	CloudProviderConfigRef string `json:"cloudProviderConfigRef"`

	// Defines the name of the providerconfig used by `crossplane-contrib/provider-kubernetes`
	// +kubebuilder:validation:Required
	ClusterProviderConfigRef string `json:"clusterProviderConfigRef"`

	// Defines the name of the cluster to map from
	// +kubebuilder:validation:Required
	ClusterName string `json:"clusterName"`

	// Defines the region or location for cloud resources
	// +kubebuilder:validation:Required
	RegionOrLocation string `json:"regionOrLocation"`
}

// XRSpec is the definition of the XR as an object
type XrSpec struct {
	XrClaimSpec `json:",inline"`
	// Defines the deletion policy for the XR
	// +optional
	DeletionPolicy string `json:"deletionPolicy"`

	// Defines a reference to the claim used by this XR
	// +optional
	ClaimRef ClaimRef `json:"claimRef"`

	// Defines the selector for the composition
	// +optional
	CompositionSelector CompositionSelector `json:"compositionSelector"`
}

// ClaimRef stores information about the claim
type ClaimRef struct {
	// The namespace the claim is stored in
	Namespace string `json:"namespace"`
}

// The selector for the composition
type CompositionSelector struct {
	MatchLabels MatchLabels `json:"matchLabels"`
}

// Labels to match on the composition for selection
type MatchLabels struct {
	// The provider label used to select the composition
	Provider string `json:"provider"`
}

// Status holds information about the status of the object
type XRStatus struct {
	// AWS holds information related to the AWS provider
	AWS Aws `json:"aws"`
}

// Aws is the  status holder for AWS
type Aws struct {
	// The list of subnets mapped by the function for AWS
	// +listType=map
	// +listMapKey=id
	Subnets []AwsSubnet `json:"subnets"`
}

// AwsSubnet is an object that holds information about a subnet defined in AWS
// +mapType=granular
type AwsSubnet struct {
	// ID The subnet ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// AvailabilityZone The availability zone this subnet is located in
	// +optional
	AvailabilityZone string `json:"availabilityZone"`

	// The Ipv4 cidr block defined for this subnet
	// +optional
	CidrBlock string `json:"cidrBlock"`

	// Is this subnet enabled for IPv6
	// +optional
	IsIpv6 bool `json:"isIpV6"`

	// The IPv6 CIDR block (if defined) for this subnet
	// +optional
	Ipv6CidrBlock string `json:"ipv6CidrBlock"`

	// A set of tags applied to this subnet
	// +optional
	// +mapType=granular
	Tags map[string]string `json:"tags"`

	// Is this a public subnet. Determined by validating an internet gateway on
	// the subnet route tables
	// +optional
	IsPublic bool `json:"isPublic"`

	// Does this subnet map public IPs to instances started in it
	// +nullable
	MapPublicIPOnLaunch *bool `json:"mapPublicIpOnLaunch,omitempty"`
}
