#!/bin/bash

docker build . -t docker.io/choclab/function-generate-subnets:v0.0.1
docker push choclab/function-generate-subnets:v0.0.1

kind delete cluster -n xfn

kind create cluster -n xfn

helm repo add crossplane https://charts.crossplane.io/master/
helm repo update
helm install crossplane --namespace crossplane --create-namespace crossplane-master/crossplane --devel

echo "Waiting for crossplane CRDs"
until grep -q functions <<<$(kubectl get crds 2>/dev/null); do
    echo -n .
    sleep 1
done
echo

kubectl config use-context kind-xfn
kubectl apply -f examples/xrender/controllers.yaml
sleep 10
kubectl apply -f examples/xrender/functions.yaml

# Wait for functions to become ready
until
    kubectl get functions function-generate-subnets -o yaml | yq '.status.conditions[] | select(.type == "Healthy" and .status == "True")' | grep -q "True" &&
        kubectl get functions function-generate-subnets -o yaml | yq '.status.conditions[] | select(.type == "Healthy" and .status == "True")' | grep -q "True" ;
do
    echo -n .
    sleep 1
done
echo

# TODO: Ammend this to point at the secret containing your credentials
kubectl create secret generic aws-credentials -n crossplane --from-literal=creds="$(awk '/\[snail\]/{x=NR+2}(NR<=x){gsub("snail", "default"); print}' ~/.aws/credentials)"

cat <<EOF | kubectl apply -f -
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: snail
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane
      name: aws-credentials
      key: creds
EOF

kubectl create namespace org-sample

kubectl apply -f examples/xrender/definition.yaml
kubectl apply -f examples/xrender/composition.yaml
sleep 10
kubectl apply -f examples/xrender/claim.yaml

