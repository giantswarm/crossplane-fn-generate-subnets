# !/bin/bash
VERSION=v0.0.3
go build . && {
    rm package/*.xpkg
    go generate ./...
    docker buildx build . -t docker.io/choclab/function-generate-subnets:${VERSION}
    crossplane xpkg build -f package --embed-runtime-image=docker.io/choclab/function-generate-subnets:${VERSION}
    crossplane xpkg push -f package/$(ls package | grep function-generate) docker.io/choclab/function-generate-subnets:${VERSION}
}
