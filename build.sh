#!/bin/bash
go generate ./...
docker build . -t docker.io/choclab/function-generate-subnets:v0.0.1
docker push choclab/function-generate-subnets:v0.0.1
