// Package main implements a Composition Function.
package main

import (
	"context"
	"math/rand"

	"github.com/alecthomas/kong"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/function-sdk-go"
	"github.com/crossplane/function-sdk-go/logging"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CLI of this Function.
type CLI struct {
	Debug bool `short:"d" help:"Emit debug logs in addition to info logs."`

	Network     string `help:"Network on which to listen for gRPC connections." default:"tcp"`
	Address     string `help:"Address at which to listen for gRPC connections." default:":9443"`
	TLSCertsDir string `help:"Directory containing server certs (tls.key, tls.crt) and the CA used to verify client certificates (ca.crt)" env:"TLS_SERVER_CERTS_DIR"`
	Insecure    bool   `help:"Run without mTLS credentials. If you supply this flag --tls-server-certs-dir will be ignored."`
	FakeClient  bool   `short:"f" help:"Run with a fake AWS client for testing locally"`
}

// FakeClient should be used for local testing
type FakeClient struct {
	ec2.Client
}

// Fakes the lookup to AWS inside the FakeClient
func (f *FakeClient) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	i := rand.Intn(2)
	var x string
	if i%2 == 0 {
		x = "igw-1a2b3c4d5e6f"
	}
	return &ec2.DescribeRouteTablesOutput{
		RouteTables: []ec2types.RouteTable{
			{
				Associations: []ec2types.RouteTableAssociation{
					{
						GatewayId: aws.String(x),
					},
				},
			},
		},
	}, nil
}

func setupFakeClient(isFake bool) {
	if !isFake {
		return
	}
	ctrl.Log.Info("Using fake client")
	getEc2Client = func(cfg aws.Config) AwsEc2Api {
		return &FakeClient{}
	}
	awsConfig = func(region, provider *string) (aws.Config, error) {
		return aws.Config{}, nil
	}
}

// Run this Function.
func (c *CLI) Run() error {
	zl := zap.New(zap.UseDevMode(c.Debug))
	log := logging.NewLogrLogger(zl.WithName(composedName))
	ctrl.SetLogger(zl)

	setupFakeClient(c.FakeClient)

	return function.Serve(&Function{log: log},
		function.Listen(c.Network, c.Address),
		function.MTLSCertificates(c.TLSCertsDir),
		function.Insecure(c.Insecure))
}

func main() {
	ctx := kong.Parse(&CLI{}, kong.Description("A Crossplane Composition Function."))
	ctx.FatalIfErrorf(ctx.Run())
}
