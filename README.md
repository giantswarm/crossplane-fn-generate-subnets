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

# Build a Docker image - see Dockerfile
$ docker build .
```

## Testing locally

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

[Crossplane]: https://crossplane.io
[Composition]: https://docs.crossplane.io/v1.13/concepts/compositions
[RunFunctionRequest]: https://github.com/crossplane/function-sdk-go/blob/a4ada4f934f6f8d3f9018581199c6c71e0343d13/proto/v1beta1/run_function.proto#L36

