# !/bin/bash
VERSION=v0.0.1-26
go build . && {
    rm package/*.xpkg
    go generate ./...
    docker buildx build . -t docker.io/choclab/function-network-discovery:${VERSION}
    crossplane xpkg build -f package --embed-runtime-image=docker.io/choclab/function-network-discovery:${VERSION}
    crossplane xpkg push -f package/$(ls package | grep function-network) docker.io/choclab/function-network-discovery:${VERSION}
}
