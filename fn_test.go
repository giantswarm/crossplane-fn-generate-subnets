package main

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/giantswarm/crossplane-fn-generate-subnets/pkg/input/v1beta1"

	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

type MockEc2ApiRtGwId struct {
	ec2.Client
}

func (m *MockEc2ApiRtGwId) DescribeRouteTables(ctx context.Context,
	params *ec2.DescribeRouteTablesInput,
	optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	x := ec2.DescribeRouteTablesOutput{
		RouteTables: []ec2types.RouteTable{
			{
				Associations: []ec2types.RouteTableAssociation{
					{
						GatewayId: aws.String("igw-bob2345"),
					},
				},
			},
		},
	}
	return &x, nil
}

type MockEc2ApiRtRouteGwId struct {
	ec2.Client
}

func (m *MockEc2ApiRtRouteGwId) DescribeRouteTables(ctx context.Context,
	params *ec2.DescribeRouteTablesInput,
	optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	x := ec2.DescribeRouteTablesOutput{
		RouteTables: []ec2types.RouteTable{
			{
				Associations: []ec2types.RouteTableAssociation{
					{
						GatewayId: nil,
					},
				},
				Routes: []ec2types.Route{
					{
						GatewayId: aws.String("igw-bob2345"),
					},
				},
			},
		},
	}
	return &x, nil
}

func TestRunFunction(t *testing.T) {

	type args struct {
		ctx context.Context
		req *fnv1beta1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1beta1.RunFunctionResponse
		err error
	}

	type mocks struct {
		ec2 func(cfg aws.Config) AwsEc2Api
		aws func(region, provider *string) (aws.Config, error)
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
		mocks  mocks
	}{
		"InputIsUndefined": {
			reason: "The Function should return a fatal response if specification is nil",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "dummy.fn.crossplane.io",
						"kind": "Input"
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1beta1.Result{
						{
							Severity: fnv1beta1.Severity_SEVERITY_FATAL,
							Message:  "object does not contain spec field",
						},
					},
				},
			},
			mocks: mocks{},
		},
		"Spec is empty": {
			reason: "The Function should return a fatal result if no input was specified",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Meta: &fnv1beta1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "dummy.fn.crossplane.io",
						"kind": "Input",
						"spec": {}
					}`),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
				},
			},
			mocks: mocks{},
		},
		"NormalResponseWaitingWhenClusterRefDoesntExist": {
			reason: "When cluster ref is undefined, we get a normal response",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Input: resource.MustStructObject(&v1beta1.Input{
						Spec: &v1beta1.Spec{
							ClusterRef: "eks-cluster",
							PatchTo:    "status.subnets",
						},
					}),
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Ttl: durationpb.New(response.DefaultTTL)},
				},
			},
			mocks: mocks{},
		},
		"NormalResponseCreateSubnets": {
			reason: "When a cluster is found and subnets are generated",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Input: resource.MustStructObject(&v1beta1.Input{
						Spec: &v1beta1.Spec{
							ClusterRef: "eks-cluster",
							PatchTo:    "status.subnets",
						},
					}),
					Observed: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete"
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									},
									"status": {
										"atProvider": {
											"vpcConfig": [
												{
													"vpcId": "vpc-12345678",
													"subnetIds": [
														"subnet-123456"
													]
												}
											]
										}
									}
								}`),
							},
						},
					},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete"
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									}
								}`),
							},
						},
					},
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete"
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
								"apiVersion": "eks.aws.upbound.io/v1beta1",
								"kind": "Cluster",
								"metadata": {
									"annotations": {
										"crossplane.io/external-name": "example",
										"crossplane.io/composition-resource-name": "eks-cluster"
									},
									"labels": {
										"crossplane.io/claim-name": "example"
									}
								},
								"managementPolicies": ["Observe"],
								"spec": {
									"forProvider": {
										"region": "eu-central-1"
									},
									"providerConfigRef": {
										"name": "example"
									},
									"writeConnectionSecretToRef": {
										"namespace": "example"
									}
								}
							}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
								"apiVersion": "ec2.aws.upbound.io/v1beta1",
								"kind": "Subnet",
								"metadata": {
									"annotations": {
										"crossplane.io/external-name": "subnet-123456"
									},
									"labels": {
										"crossplane.io/claim-name": "example"
									},
									"name": "test-subnet-123456"
								},
								"spec": {
									"managementPolicies": ["Observe"],
									"forProvider": {
										"region": "eu-central-1",
										"vpcId": "vpc-12345678"
									},
									"providerConfigRef": {
										"name": "example"
									},
									"writeConnectionSecretToRef": {
										"name": "subnet-123456",
										"namespace": "example"
									}
								}
							}`),
							},
						},
					},
				},
			},
			mocks: mocks{},
		},
		"NormalResponsePatchXR": {
			reason: "When a cluster is found and subnets are generated",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Input: resource.MustStructObject(&v1beta1.Input{
						Spec: &v1beta1.Spec{
							ClusterRef: "eks-cluster",
							PatchTo:    "status.subnets",
						},
					}),
					Observed: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									},
									"status": {
										"atProvider": {
											"vpcConfig": [
												{
													"vpcId": "vpc-12345678",
													"subnetIds": [
														"subnet-123456"
													]
												}
											]
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										},
										"name": "subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									},
									"status": {
										"atProvider": {
											"id": "subnet-123456",
											"availabilityZone": "eu-central-1a",
											"cidrBlock": "192.168.0.0/8",
											"isIpV6": false,
											"ipv6CidrBlock": "",
											"tags": {},
											"mapPublicIpOnLaunch": false
										}
									}
								}`),
							},
						},
					},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										},
										"name": "test-subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									}
								}`),
							},
						},
					},
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								},
								"status": {
									"subnets": [
										{
											"id": "subnet-123456",
											"availabilityZone": "eu-central-1a",
											"cidrBlock": "192.168.0.0/8",
											"isIpV6": false,
											"ipv6CidrBlock": "",
											"isPublic": true,
											"tags": {}
										}
									]
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example",
											"test": "label",
											"anothertest": "label"
										},
										"name": "test-subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									}
								}`),
							},
						},
					},
				},
			},
			mocks: mocks{
				ec2: func(aws.Config) AwsEc2Api {
					return &MockEc2ApiRtGwId{}
				},
				aws: func(region, provider *string) (aws.Config, error) {
					return aws.Config{}, nil
				},
			},
		},
		"NormalResponsePatchXRRouteTableRoute": {
			reason: "When a cluster is found and subnets are generated",
			args: args{
				req: &fnv1beta1.RunFunctionRequest{
					Input: resource.MustStructObject(&v1beta1.Input{
						Spec: &v1beta1.Spec{
							ClusterRef: "eks-cluster",
							PatchTo:    "status.subnets",
						},
					}),
					Observed: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									},
									"status": {
										"atProvider": {
											"vpcConfig": [
												{
													"vpcId": "vpc-12345678",
													"subnetIds": [
														"subnet-123456"
													]
												}
											]
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										},
										"name": "subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									},
									"status": {
										"atProvider": {
											"id": "subnet-123456",
											"availabilityZone": "eu-central-1a",
											"cidrBlock": "192.168.0.0/8",
											"isIpV6": false,
											"ipv6CidrBlock": "",
											"tags": {},
											"mapPublicIpOnLaunch": false
										}
									}
								}`),
							},
						},
					},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										},
										"name": "test-subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									}
								}`),
							},
						},
					},
				},
			},
			want: want{
				rsp: &fnv1beta1.RunFunctionResponse{
					Meta: &fnv1beta1.ResponseMeta{Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1beta1.State{
						Composite: &fnv1beta1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion":"example.org/v1",
								"kind":"XR",
								"spec": {
									"clusterName": "test",
									"clusterProviderConfigRef": "stringy",
									"regionOrLocation": "placey",
									"deletionPolicy": "Delete",
									"labels": {
										"test": "label",
										"anothertest": "label"
									}
								},
								"status": {
									"subnets": [
										{
											"id": "subnet-123456",
											"availabilityZone": "eu-central-1a",
											"cidrBlock": "192.168.0.0/8",
											"isIpV6": false,
											"ipv6CidrBlock": "",
											"isPublic": true,
											"tags": {}
										}
									]
								}
							}`),
						},
						Resources: map[string]*fnv1beta1.Resource{
							"eks-cluster": {
								Resource: resource.MustStructJSON(`{
									"apiVersion": "eks.aws.upbound.io/v1beta1",
									"kind": "Cluster",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "example",
											"crossplane.io/composition-resource-name": "eks-cluster"
										},
										"labels": {
											"crossplane.io/claim-name": "example"
										}
									},
									"managementPolicies": ["Observe"],
									"spec": {
										"forProvider": {
											"region": "eu-central-1"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"namespace": "example"
										}
									}
								}`),
							},
							"crossplane-fn-generate-subnets-subnet-123456": {
								Ready: fnv1beta1.Ready_READY_TRUE,
								Resource: resource.MustStructJSON(`{
									"apiVersion": "ec2.aws.upbound.io/v1beta1",
									"kind": "Subnet",
									"metadata": {
										"annotations": {
											"crossplane.io/external-name": "subnet-123456"
										},
										"labels": {
											"crossplane.io/claim-name": "example",
											"test": "label",
											"anothertest": "label"
										},
										"name": "test-subnet-123456"
									},
									"spec": {
										"managementPolicies": ["Observe"],
										"forProvider": {
											"region": "eu-central-1",
											"vpcId": "vpc-12345678"
										},
										"providerConfigRef": {
											"name": "example"
										},
										"writeConnectionSecretToRef": {
											"name": "subnet-123456",
											"namespace": "example"
										}
									}
								}`),
							},
						},
					},
				},
			},
			mocks: mocks{
				ec2: func(aws.Config) AwsEc2Api {
					return &MockEc2ApiRtRouteGwId{}
				},
				aws: func(region, provider *string) (aws.Config, error) {
					return aws.Config{}, nil
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			getEc2Client = tc.mocks.ec2
			awsConfig = tc.mocks.aws

			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)
			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestFakeClient(t *testing.T) {
	setupFakeClient(true)
	aws, _ := awsConfig(aws.String("eu-central-1"), aws.String("fakeProvider"))
	api := getEc2Client(aws)
	result, _ := DescribeRouteTables(context.TODO(), api, nil)
	if *result.RouteTables[0].Associations[0].GatewayId != "igw-1a2b3c4d5e6f" {
		t.Error("fake client failed")
	}
}
