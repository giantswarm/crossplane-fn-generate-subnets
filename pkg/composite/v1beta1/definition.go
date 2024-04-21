// Package v1beta1 contains the definition of the XR requirements for using this function
// +kubebuilder:object:generate=true
// +groupName=networkdiscovery.fn.giantswarm.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AWS is an object that holds VPCs
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane;composition

type Aws struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The VPCs defined in this AWS account
	// +mapType=granular
	Vpcs map[string]Vpc `json:"vpcs"`
}

// Vpc holds VPC information
type Vpc struct {
	// ID The VPC ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// The Ipv4 cidr block defined for this VPC
	// +optional
	CidrBlock string `json:"cidrBlock"`

	// A map of public subnets defined in this VPC
	// +kubebuilder:validation:Required
	// +mapType=granular
	PublicSubnets map[string]string `json:"publicSubnets"`

	// A map of private subnets defined in this VPC
	// +kubebuilder:validation:Required
	// +mapType=granular
	PrivateSubnets map[string]string `json:"privateSubnets"`

	// A map of public route tables defined in this VPC
	// +kubebuilder:validation:Required
	// +mapType=granular
	PublicRouteTables map[string]string `json:"publicRouteTables"`

	// A map of private route tables defined in this VPC
	// +kubebuilder:validation:Required
	// +mapType=granular
	PrivateRouteTables map[string]string `json:"privateRouteTables"`

	// The internet gateway defined in this VPC
	// +optional
	InternetGateway string `json:"internetGateway"`

	// A map of NAT gateways defined in this VPC
	// +mapType=granular
	// +optional
	NatGateways map[string]string `json:"natGateways"`

	// A map of transit gateways defined in this VPC
	// +optional
	TransitGateways map[string]string `json:"transitGateways"`

	// A map of VPC peering connections defined in this VPC
	// +mapType=granular
	// +optional
	VpcPeeringConnections map[string]string `json:"vpcPeeringConnections"`

	// A map of security groups defined in this VPC
	// +kubebuilder:validation:Required
	// +mapType=granular
	SecurityGroups map[string]string `json:"securityGroups"`
}

// AwsSubnet is an object that holds information about a subnet defined in AWS
// +mapType=granular
type Subnet struct {
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

	// Is this a public subnet. Determined by validating an internet gateway on
	// the subnet route tables
	// +optional
	IsPublic bool `json:"isPublic"`

	// Does this subnet map public IPs to instances started in it
	// +nullable
	MapPublicIPOnLaunch *bool `json:"mapPublicIpOnLaunch,omitempty"`

	// The route tables associated with this subnet
	// +mapType=granular
	RouteTables map[string]RouteTable `json:"routeTables"`

	// The internet gateway associated with this subnet
	// +optional
	InternetGateway string `json:"internetGateway"`

	// A map of NAT gateways associated with this subnet
	// +mapType=granular
	// +optional
	NatGateways map[string]string `json:"natGateways"`

	// A map of transit gateways associated with this subnet
	// +mapType=granular
	// +optional
	TransitGateways map[string]string `json:"transitGateways"`

	// A map of VPC peering connections associated with this subnet
	// +mapType=granular
	// +optional
	VpcPeeringConnections map[string]string `json:"vpcPeeringConnections"`
}

// AwsRouteTable is an object that holds information about a route table defined in AWS
// +mapType=granular
type RouteTable struct {
	// ID The route table ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// The name of the route table
	// +optional
	Name string `json:"name"`

	// Is this a public route table. Determined by validating the
	// existence of an internet gateway
	// +optional
	IsPublic bool `json:"isPublic"`

	// The routes defined in this route table
	// +mapType=granular
	Routes map[string]Route `json:"routes"`

	// The associations defined for this route table
	// +listType=map
	// +listMapKey=id
	Associations []Association `json:"associations"`
}

// AwsRoute is an object that holds information about a route defined in AWS
// +mapType=granular
type Route struct {
	// ID The route ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// The destination CIDR block for this route
	// +optional
	DestinationCidrBlock string `json:"destinationCidrBlock"`

	// The gateway ID for this route
	// +optional
	GatewayID string `json:"gatewayId"`

	// The instance ID for this route
	// +optional
	InstanceID string `json:"instanceId"`

	// The NAT gateway ID for this route
	// +optional
	NatGatewayID string `json:"natGatewayId"`

	// The network interface ID for this route
	// +optional
	NetworkInterfaceID string `json:"networkInterfaceId"`

	// The transit gateway ID for this route
	// +optional
	TransitGateway string `json:"transitGateway"`

	// The VPC peering connection ID for this route
	// +optional
	VpcPeeringConnectionID string `json:"vpcPeeringConnectionId"`

	// The local gateway ID for this route
	// +optional
	LocalGatewayID string `json:"localGatewayId"`

	// The carrier gateway ID for this route
	// +optional
	CarrierGatewayID string `json:"carrierGatewayId"`

	// The prefix list ID for this route
	// +optional
	PrefixListID string `json:"prefixListId"`

	// The egress only internet gateway ID for this route
	// +optional
	EgressOnlyInternetGatewayID string `json:"egressOnlyInternetGatewayId"`

	// The destination IPv6 CIDR block for this route
	// +optional
	DestinationIpv6CidrBlock string `json:"destinationIpv6CidrBlock"`
}

// AwsAssociation is an object that holds information about an association defined in AWS
// +mapType=granular
type Association struct {
	// ID The association ID
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// The association state
	// +optional
	State string `json:"state"`

	// The association main flag
	// +optional
	Main bool `json:"main"`

	// The association route table ID
	// +optional
	RouteTableID string `json:"routeTableId"`

	// The association subnet ID
	// +optional
	SubnetID string `json:"subnetId"`
}
