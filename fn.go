package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/composed"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/crossplane/function-template-go/input/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const composedName = "function-subnets"

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag", req.GetMeta().GetTag())
	rsp := response.To(req, response.DefaultTTL)
	var err error

	in := &v1beta1.Input{}
	if err = request.GetInput(req, in); err != nil || in.Spec == nil {
		if err == nil {
			err = fmt.Errorf("spec is nil")
		}
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get observed composite resource"))
		return rsp, nil
	}

	xr, err := request.GetDesiredCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composed resources from %T", req))
		return rsp, nil
	}

	xr.Resource.SetAPIVersion(oxr.Resource.GetAPIVersion())
	xr.Resource.SetKind(oxr.Resource.GetKind())

	existing, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composed resources from %T", req))
		return rsp, nil
	}
	f.log.Debug("Existing", "", existing)

	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired composite resources from %T", req))
		return rsp, nil
	}

	var (
		cluster resource.ObservedComposed
		ok      bool
	)
	if cluster, ok = existing[in.Spec.ClusterRef]; !ok {
		response.Normal(rsp, "Waiting for resource")
		return rsp, nil
	}
	var object ClusterObject
	str, _ := json.Marshal(cluster.Resource.Object)
	if err := json.Unmarshal(str, &object); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "Failed to unmarshal cluster object"))
		return rsp, nil
	}
	f.log.Debug("Got", "Object", object)

	var vpcs []VpcConfig = make([]VpcConfig, 0)
	if object.Status.AtProvider != nil {
		vpcs = object.Status.AtProvider.VpcConfig
	}

	var subnetDetails []interface{} = make([]interface{}, 0)
	for _, vpc := range vpcs {
		for _, subnetID := range vpc.Subnets {
			subnetID := subnetID
			var name resource.Name = resource.Name(fmt.Sprintf("%s-%s", composedName, subnetID))
			if subnet, ok := existing[name]; ok {
				var sn SubnetObject
				str, _ := json.Marshal(subnet.Resource.Object)
				if err := json.Unmarshal(str, &sn); err != nil {
					continue
				}

				if (sn.Status).AtProvider != nil {
					subnetDetails = append(subnetDetails, f.subnetToCapiStruct(sn.Status.AtProvider))
				}
			}

			f.addDesiredTo(
				desired,
				name,
				object.Spec.ProviderConfig.Name,
				object.Spec.ConnectionSecret.Namespace,
				object.Spec.ForProvider.Region,
				subnetID,
				vpc.VpcID,
				object.Metadata,
			)
		}
	}
	f.log.Debug(string(in.Spec.ClusterRef), "Subnets", subnetDetails)

	if err = f.patchFieldValueToObject(in.Spec.PatchTo, subnetDetails, xr.Resource); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot render ToComposite patches for composed resource %q", in.Spec.PatchTo))
		return rsp, nil
	}

	if err = response.SetDesiredCompositeResource(rsp, xr); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composite resources in %T", rsp))
		return rsp, nil
	}

	if err = response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	response.Normal(rsp, "Successful run")
	return rsp, nil
}

func (f *Function) subnetToCapiStruct(subnet *Subnet) map[string]interface{} {
	var (
		value map[string]interface{}
	)

	// you can't really trust `mapPublicIpOnLaunch` but there isn't a better option
	// without performing a lookup on the route table however without the route table ID
	// we've no way at the moment of filtering for this subnet
	// so this is quick and dirty
	if _, ok := (*subnet).Tags["giantswarm.io/public"]; ok || (subnet.MapPublicIPOnLaunch != nil && *(subnet.MapPublicIPOnLaunch)) {
		subnet.IsPublic = true
	}

	str, _ := json.Marshal(subnet)
	f.log.Debug(string(str))
	if err := json.Unmarshal(str, &value); err == nil {
		delete(value, "mapPublicIpOnLaunch")
	}
	return value
}

func (f *Function) patchFieldValueToObject(fieldPath string, value []interface{}, to runtime.Object) error {
	paved, err := fieldpath.PaveObject(to)
	if err != nil {
		return err
	}

	if err := paved.SetValue(fieldPath, value); err != nil {
		return err
	}

	return runtime.DefaultUnstructuredConverter.FromUnstructured(paved.UnstructuredContent(), to)
}

func (f *Function) addDesiredTo(
	desired map[resource.Name]*resource.DesiredComposed,
	name resource.Name,
	providerConfig,
	connectionSecretNamespace,
	region,
	subnet,
	vpcID string,
	metadata Metadata,
) {
	var clusterName string = metadata.Annotations["crossplane.io/external-name"]
	f.log.Info("Adding", "subnet", subnet, "Cluster", clusterName)

	objectSpec := map[string]interface{}{
		"apiVersion": "ec2.aws.upbound.io/v1beta1",
		"kind":       "Subnet",
		"metadata": map[string]interface{}{
			"name": subnet,
			"annotations": map[string]interface{}{
				"crossplane.io/external-name": subnet,
			},
			"labels": metadata.Labels,
		},
		"spec": map[string]interface{}{
			"providerConfigRef": map[string]interface{}{
				"name": providerConfig,
			},
			"managementPolicies": []interface{}{"Observe"},
			"forProvider": map[string]interface{}{
				"region": region,
				"vpcId":  vpcID,
			},
			"writeConnectionSecretToRef": map[string]interface{}{
				"name":      subnet,
				"namespace": connectionSecretNamespace,
			},
		},
	}

	desired[name] = &resource.DesiredComposed{
		Resource: &composed.Unstructured{
			Unstructured: unstructured.Unstructured{
				Object: objectSpec,
			},
		},
		Ready: resource.ReadyTrue,
	}
}
