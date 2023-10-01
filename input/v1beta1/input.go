// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=generatesubnets.fn.giantswarm.io
// +versionName=v1beta1
package v1beta1

import (
	"github.com/crossplane/function-sdk-go/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// Input can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type Input struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Defines the spec for this input
	Spec *Spec `json:"spec,omitempty"`
}

// Spec - Defines the spec given to this input type, providing the required, and optional elements that may be defined
type Spec struct {
	// ClusterRef The XR name of the cluster
	ClusterRef resource.Name `json:"clusterRef"`

	// PatchTo specified the path to apply the subnets patch
	PatchTo string `json:"patchTo"`
}
