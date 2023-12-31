package main

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	xfnaws "github.com/giantswarm/xfnlib/pkg/auth/aws"
)

// EC2API Describes the functions required to access data on the AWS EC2 api
type AwsEc2Api interface {
	DescribeRouteTables(ctx context.Context,
		params *ec2.DescribeRouteTablesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
}

// Get the EC2 Launch template versions for a given launch template
func DescribeRouteTables(c context.Context, api AwsEc2Api, input *ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error) {
	return api.DescribeRouteTables(c, input)
}

var (
	getEc2Client = func(cfg aws.Config) AwsEc2Api {
		return ec2.NewFromConfig(cfg)
	}

	awsConfig = func(region, provider *string) (aws.Config, error) {
		return xfnaws.Config(region, provider)
	}
)

// FindAWSPublicRouteTables looks for the route tables attached to a given
// subnet
//
// This method checks the ec2 api for route tables attached to a subnet.
// Once discovered, it looks to see if there is an internet gateway attached
// to that route table, returning true if that is the case marking this as a
// public subnet
func (f *Function) FindAWSPublicRouteTables(subnetId, region, providerConfig *string) (bool, error) {
	var (
		err        error
		cfg        aws.Config
		filterName string = "association.subnet-id"
		rtbls      *ec2.DescribeRouteTablesOutput
		input      ec2.DescribeRouteTablesInput
		filters    = make([]ec2types.Filter, 0)
	)

	filters = append(filters, ec2types.Filter{
		Name: &filterName,
		Values: []string{
			*subnetId,
		},
	})

	input = ec2.DescribeRouteTablesInput{
		Filters: filters,
	}
	// Set up the assume role clients
	if cfg, err = awsConfig(region, providerConfig); err != nil {
		err = errors.Wrap(err, "failed to load aws config for assume role")
		return false, err
	}

	ec2client := getEc2Client(cfg)
	if rtbls, err = DescribeRouteTables(context.TODO(), ec2client, &input); err != nil {
		err = errors.Wrap(err, "failed to load aws route tables for subnet "+*subnetId)
		return false, err
	}

	for _, rt := range rtbls.RouteTables {
		for _, assoc := range rt.Associations {
			f.log.Debug("route tables", "assoc", assoc)
			if assoc.GatewayId != nil && strings.HasPrefix(*assoc.GatewayId, "igw-") {
				return true, nil
			}
			for _, r := range rt.Routes {
				f.log.Debug("routes ", "route", r)
				if r.GatewayId != nil && strings.HasPrefix(*r.GatewayId, "igw-") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
