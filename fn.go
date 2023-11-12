package main

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	fnc "github.com/giantswarm/crossplane-fn-generate-subnets/pkg/composite/v1beta1"
	"github.com/giantswarm/crossplane-fn-generate-subnets/pkg/input/v1beta1"
	"github.com/giantswarm/xfnlib/pkg/composite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const composedName = "crossplane-fn-generate-subnets"

// RunFunction runs the composition Function to generate subnets from the given cluster
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (rsp *fnv1beta1.RunFunctionResponse, err error) {
	f.log.Info("preparing function", composedName, req.GetMeta().GetTag())
	rsp = response.To(req, response.DefaultTTL)

	type subnetType struct {
		objectSpec unstructured.Unstructured
		subnetId   string
	}

	var (
		composed      *composite.Composition
		compositeXr   fnc.CompositeObject
		input         v1beta1.Input
		cluster       resource.ObservedComposed
		object        ClusterObject
		ok            bool
		vpcs          []VpcConfig                  = make([]VpcConfig, 0)
		subnetDetails []interface{}                = make([]interface{}, 0)
		subnetsToAdd  map[resource.Name]subnetType = make(map[resource.Name]subnetType)
		count         int                          = 0
	)

	if composed, err = composite.New(req, &input, &compositeXr); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "error setting up function "+composedName))
		return rsp, nil
	}

	if input.Spec == nil {
		response.Fatal(rsp, &composite.MissingSpec{})
		return rsp, nil
	}

	if cluster, ok = composed.ObservedComposed[input.Spec.ClusterRef]; !ok {
		response.Normal(rsp, "waiting for resource")
		return rsp, nil
	}

	if err = composite.To(cluster.Resource.Object, &object); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "Failed to unmarshal cluster object"))
		return rsp, nil
	}

	f.log.Debug("RunFunction", "Cluster Object", object)

	if object.Status.AtProvider != nil {
		vpcs = object.Status.AtProvider.VpcConfig
	}

	var labels map[string]interface{} = make(map[string]interface{})
	{
		for k, v := range object.Metadata.Labels {
			labels[k] = v
		}

		for k, v := range compositeXr.Spec.Labels {
			labels[k] = v
		}
	}

	for _, vpc := range vpcs {
		count += len(vpc.Subnets)
		for _, subnetID := range vpc.Subnets {
			var name resource.Name = resource.Name(fmt.Sprintf("%s-%s", composedName, subnetID))
			if subnet, ok := composed.ObservedComposed[name]; ok {
				var sn AwsSubnetObject
				if err := composite.To(subnet.Resource.Object, &sn); err != nil {
					f.log.Info(err.Error())
					continue
				}

				if (sn.Status).AtProvider != nil {
					var (
						details  map[string]interface{}
						region   *string = &compositeXr.Spec.RegionOrLocation
						provider *string = &compositeXr.Spec.CloudProviderConfigRef
					)
					if details, err = f.subnetToCapaStruct(sn.Status.AtProvider, region, provider); err != nil {
						f.log.Info(err.Error())
						continue
					}
					subnetDetails = append(subnetDetails, details)
				}
			}

			f.log.Info("Adding subnet to provider list", "subnet", subnetID)
			objectSpec := unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "ec2.aws.upbound.io/v1beta1",
					"kind":       "Subnet",
					"metadata": map[string]interface{}{
						"name": fmt.Sprintf("%s-%s", compositeXr.Spec.ClusterName, subnetID),
						"annotations": map[string]interface{}{
							"crossplane.io/external-name": subnetID,
						},
						"labels": labels,
					},
					"spec": map[string]interface{}{
						"providerConfigRef": map[string]interface{}{
							"name": object.Spec.ProviderConfig.Name,
						},
						"managementPolicies": []interface{}{"Observe"},
						"forProvider": map[string]interface{}{
							"region": object.Spec.ForProvider.Region,
							"vpcId":  vpc.VpcID,
						},
						"writeConnectionSecretToRef": map[string]interface{}{
							"name":      subnetID,
							"namespace": object.Spec.ConnectionSecret.Namespace,
						},
					},
				},
			}

			subnetsToAdd[name] = subnetType{
				subnetId:   subnetID,
				objectSpec: objectSpec,
			}
		}
	}

	// We want to prevent adding subnets unless all of them have been
	// reconciled from the cloud provider. This helps avoid transient
	// errors
	if len(subnetsToAdd) == count {
		for name, item := range subnetsToAdd {
			if err = composed.AddDesired(string(name), &item.objectSpec); err != nil {
				f.log.Debug("RunFunction", "subnet", &item.subnetId, "object", item.objectSpec, "  #  err", err)
				continue
			}
		}
	}

	// Similar to the above, we won't populate the XR status unless we've
	// successfully reconciled all subnet status's.
	// This blocks the creation of CAPI resources until we have all the
	// details fully reconciled ensuring we don't hand incomplete data
	// to the CAPI providers
	f.log.Debug(string(input.Spec.ClusterRef), "Subnets", subnetDetails)
	if len(subnetDetails) == count {
		// Don't patch unless we have a populated array
		if err = f.patchFieldValueToObject(input.Spec.PatchTo, subnetDetails, composed.DesiredComposite.Resource); err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot render ToComposite patch %q", input.Spec.PatchTo))
			return rsp, nil
		}
	}

	if err = composed.ToResponse(rsp); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot convert composition to response %T", rsp))
		return
	}

	response.Normal(rsp, "Successful run")
	return rsp, nil
}

// subnetToCapaStruct takes a subnet object and converts this to a
// type that can be used within the AWSManagedControlPlane object
func (f *Function) subnetToCapaStruct(subnet *fnc.AwsSubnet, region, provider *string) (map[string]interface{}, error) {
	var (
		value  map[string]interface{}
		err    error
		public bool
	)

	if public, err = f.FindAWSPublicRouteTables(&subnet.ID, region, provider); err != nil {
		return nil, err
	}

	subnet.IsPublic = public

	if err := composite.To(subnet, &value); err == nil {
		delete(value, "mapPublicIpOnLaunch")
	}

	return value, nil
}

// patchFieldValueToObject is used to push information onto the XR status
func (f *Function) patchFieldValueToObject(fieldPath string, value []interface{}, to runtime.Object) (err error) {
	var paved *fieldpath.Paved
	if paved, err = fieldpath.PaveObject(to); err != nil {
		return
	}

	if err = paved.SetValue(fieldPath, value); err != nil {
		return
	}

	return runtime.DefaultUnstructuredConverter.FromUnstructured(paved.UnstructuredContent(), to)
}
