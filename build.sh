#!/bin/bash
go build . && {
    rm package/*.xpkg
    go generate ./...
    docker buildx build . -t docker.io/choclab/function-generate-subnets:v0.0.1-dev
    crossplane xpkg build -f package --embed-runtime-image=docker.io/choclab/function-generate-subnets:v0.0.1
    crossplane xpkg push -f package/$(ls package | grep function-generate) docker.io/choclab/function-generate-subnets:v0.0.1
}
