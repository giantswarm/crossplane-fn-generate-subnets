# function-generate-subnets

A [Crossplane] Composition Function which will generate an ObserveOnly subnet
object for each subnet ID found in an EKS cluster resource, then patch specific
information from those objects to a field on the composite resource.

## How it works

> **Warning**
> This plugin is requires Crossplane v1.14 which is currently unreleased
> (due 1st November 2023).
>
> The example composition is also written for Crossplane v1.14 and will
> not work on any current MC version.
> 
> To support this, the script [`kind.sh`](./kind.sh) is provided to
> help you understand how this works by spinning crossplane up inside a
> kind cluster for local development.

In order to use this function as part of the [Composition], the composition 
must be written to use pipeline mode. This is a (currently undocumented)
mode for compositions.

```yaml
spec:
  compositeTypeRef:
    apiVersion: crossplane.giantswarm.io/v1alpha1
    kind: CompositeEksImport
  mode: Pipeline
  pipeline:
  - step: collect-cluster
    ...
  - step: generate-subnets
    ...
```

> **Note**
> To run the kind cluster, you need to open the kind.sh script and
> edit the TODO for the kube secret to point at the credentials you have for
> the cluster.
>
> Specifically the line to edit is:
>
> ```bash
> kubectl create secret generic aws-credentials -n crossplane \
>   --from-literal=creds="$(awk '/\[snail\]/{x=NR+2}(NR<=x){gsub("snail", "default"); print}' ~/.aws/credentials)"
> ```
>
> Where I have a aws/credentials entry for `snail` that I convert to `default` for the provider.

**Example:**

<details>

<summary>composition.yaml</summary>

```yaml
  - step: generate-subnets
    functionRef:
      name: function-generate-subnets
    input:
      apiVersion: generator.fn.giantswarm.io
      kind: Subnet
      metadata:
        namespace: crossplane
      spec:
        clusterRef: eks-cluster
        patchTo: status.subnets
```

</details>

<details>

<summary>status.subnets</summary>

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
  package: choclab/function-generate-subnets:v0.0.1
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

To test this application locally you need to first install [xrender].

In one window, execute:

```bash
go run . --insecure -d
```

whilst in a second window, run:

```bash
xrender examples/xrender/xr.yaml examples/xrender/composition.yaml examples/xrender/functions.yaml -o examples/xrender/observed.yaml
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
[Composition]: https://docs.crossplane.io/v1.13/concepts/compositions
[RunFunctionRequest]: https://github.com/crossplane/function-sdk-go/blob/a4ada4f934f6f8d3f9018581199c6c71e0343d13/proto/v1beta1/run_function.proto#L36
[xrender]: https://github.com/crossplane-contrib/xrender