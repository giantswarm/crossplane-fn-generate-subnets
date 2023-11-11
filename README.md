# function-generate-subnets

A [Crossplane] Composition Function which will generate an ObserveOnly subnet
object for each subnet ID found in an EKS cluster resource, then patch specific
information from those objects to a field on the composite resource.

In order to use this function as part of the [Composition], the composition
must be written to use pipeline mode. See the documentation on 
[Composition functions] to better understand how this is integrated

## Composition integration

This function is placed in the pipeline with a reference to the cluster object
for that composition and an additional reference of where to patch information
about the subnets it is generating for that provider.

This should be specified in your composition as:

```yaml
  - step: generate-subnets
    functionRef:
      name: function-generate-subnets
    input:
      apiVersion: generatesubnets.fn.giantswarm.io
      kind: Subnet
      metadata:
        namespace: crossplane
      spec:
        clusterRef: eks-cluster
        patchTo: status.subnets
```

The function requires information from the XR and is opinionated about how that
information should be presented.

In order to support the integration of this function into your pipelines,
details about the XR requirements are published in 
[definition_xrs.yaml](./package/composite/definition_xrobjectdefinitions.yaml).

The relevant sections of this XR specification should be extracted and merged
into the definition you are writing. The specification cannot be used 
independantly as an XRD as it does not define all the required elements of an 
XRD, just the custom fields required by this function.

## How it works

This function reads the subnets from a cluster object relevant to the cloud
provider.

> For each integrated type, only the official upbound providers are supported.

It then uses the subnet IDs to generate one `Subnet` object for each subnet in
the list and adds these to the set of desired composed objects and allows them
to reconcile.

Once reconciliation is complete, the function collects information from the
`status.atProvider` for each subnet and compiles this into a list of objects
which is then applied to the status field of the XR.

As part of the reconciliation, the function calls out to the cloud provider to
retrieve additional information such as whether the subnet has an internet
gateway attached which the function uses to determine whether the subnet is
public or private.

### Providers currently supported by this function

- EKS cluster `clusters.eks.upbound.io`

### Example status patch output

<details>

<summary>status.aws.subnets</summary>

```yaml
    subnets:
    - availabilityZone: eu-central-1c
      cidrBlock: 192.168.128.0/19
      id: subnet-11111111111111111
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: false
      tags: {}
    - availabilityZone: eu-central-1b
      cidrBlock: 192.168.64.0/19
      id: subnet-22222222222222222
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: true
      tags: {}
    - availabilityZone: eu-central-1b
      cidrBlock: 192.168.160.0/19
      id: subnet-33333333333333333
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: false
      tags: {}
    - availabilityZone: eu-central-1a
      cidrBlock: 192.168.96.0/19
      id: subnet-44444444444444444
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: false
      tags: {}
    - availabilityZone: eu-central-1c
      cidrBlock: 192.168.32.0/19
      id: subnet-555555555555555555
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: true
      tags: {}
    - availabilityZone: eu-central-1a
      cidrBlock: 192.168.0.0/19
      id: subnet-6666666666666666666
      ipv6CidrBlock: ""
      isIpV6: false
      isPublic: true
      tags: {}
```

</details>

## Installing

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-generate-subnets
spec:
  package: docker.io/giantswarm/crossplane-fn-generate-subnets:v0.1.0
```

## Building

```shell
# Run code generation - see input/generate.go
$ go generate ./...

# Lint the code
$ docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/v1.54.2:/root/.cache -w /app golangci/golangci-lint:v1.54.2 golangci-lint run

$ go test -cover ./...
?       github.com/crossplane/function-template-go/input/v1beta1        [no test files]
ok      github.com/crossplane/function-template-go      0.006s  coverage: 71.1% of statements

# Build a Docker image - see Dockerfile
$ docker build .
```

## Testing locally

To test this application locally you need to first install [crossplane-cli].

In one window, execute:

```bash
go run . --insecure -d
```

whilst in a second window, run:

```bash
crossplane beta render examples/xrender/xr.yaml examples/xrender/composition.yaml examples/xrender/functions.yaml -o examples/xrender/observed.yaml
```

## Known Issues

It is just a prototype for now, not ready for production use.

- `upbound-provider-aws-ec2:v0.40.0` has a bug whereby the finalizer is not set on the subnets. This needs to be added as part of the generation
- `upbound-provider-aws-eks:v0.41.0` has a bug whereby the provider will initially error with `Transient config error`.
  This eventually goes away but takes ~10m for the full set to reconcile.

It is currently best to use the following combination of providers:

- `upbound-provider-aws-ec2:v0.41.0`
- `upbound-provider-aws-eks:v0.40.0`

Upstream issue tracking EKS provider bug: [https://github.com/upbound/provider-aws/issues/904](https://github.com/upbound/provider-aws/issues/904)

[Crossplane]: https://crossplane.io
[crossplane-cli]: https://github.com/crossplane/crossplane/releases/tag/v1.14.0-rc.1
[Composition]: https://docs.crossplane.io/v1.13/concepts/compositions
[Composition functions]: https://docs.crossplane.io/latest/concepts/compositions/#use-composition-functions
[RunFunctionRequest]: https://github.com/crossplane/function-sdk-go/blob/a4ada4f934f6f8d3f9018581199c6c71e0343d13/proto/v1beta1/run_function.proto#L36
[xrender]: https://github.com/crossplane-contrib/xrender
