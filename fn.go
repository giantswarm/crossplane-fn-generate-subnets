package main

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/giantswarm/crossplane-fn-generate-subnets/input/v1beta1"
	"github.com/giantswarm/xfnlib/pkg/composite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const composedName = "crossplane-fn-generate-subnets"

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (rsp *fnv1beta1.RunFunctionResponse, err error) {
	f.log.Info("preparing function", composedName, req.GetMeta().GetTag())
	rsp = response.To(req, response.DefaultTTL)

	input := v1beta1.Input{}
	if f.composed, err = composite.New(req, &input, &f.composite); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "error setting up function "+composedName))
		return rsp, nil
	}

	var (
		cluster       resource.ObservedComposed
		object        ClusterObject
		ok            bool
		vpcs          []VpcConfig   = make([]VpcConfig, 0)
		subnetDetails []interface{} = make([]interface{}, 0)
	)

	if cluster, ok = f.composed.ObservedComposed[input.Spec.ClusterRef]; !ok {
		response.Normal(rsp, "Waiting for resource")
		return rsp, nil
	}

	if err = composite.To(cluster.Resource.Object, &object); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "Failed to unmarshal cluster object"))
		return rsp, nil
	}

	f.log.Debug("Got", "Object", object)

	if object.Status.AtProvider != nil {
		vpcs = object.Status.AtProvider.VpcConfig
	}

	type subnetType struct {
		objectSpec unstructured.Unstructured
		subnetId   string
	}
	var (
		subnetsToAdd map[resource.Name]subnetType = make(map[resource.Name]subnetType)
		count        int                          = 0
	)
	for _, vpc := range vpcs {
		count += len(vpc.Subnets)
		for _, subnetID := range vpc.Subnets {
			subnetID := subnetID
			var name resource.Name = resource.Name(fmt.Sprintf("%s-%s", composedName, subnetID))
			if subnet, ok := f.composed.ObservedComposed[name]; ok {
				var sn SubnetObject
				if err := composite.To(subnet.Resource.Object, &sn); err != nil {
					f.log.Info(err.Error())
					continue
				}

				if (sn.Status).AtProvider != nil {
					var details map[string]interface{}
					if details, err = f.subnetToCapaStruct(sn.Status.AtProvider, &object.Spec.ForProvider.Region, &object.Spec.ProviderConfig.Name); err != nil {
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
						"name": fmt.Sprintf("%s-%s", f.composite.Spec.ClusterName, subnetID),
						"annotations": map[string]interface{}{
							"crossplane.io/external-name": subnetID,
						},
						"labels": object.Metadata.Labels,
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

	if len(subnetsToAdd) == count {
		for name, item := range subnetsToAdd {
			if err = f.composed.AddDesired(string(name), &item.objectSpec); err != nil {
				f.log.Info("Failed to add desired object", "subnet", &item.subnetId, "object", item.objectSpec, "  #  err", err)
				continue
			}
		}
	}

	f.log.Debug(string(input.Spec.ClusterRef), "Subnets", subnetDetails)
	if len(subnetDetails) == count {
		// Don't patch unless we have a populated array
		if err = f.patchFieldValueToObject(input.Spec.PatchTo, subnetDetails, f.composed.DesiredComposite.Resource); err != nil {
			response.Fatal(rsp, errors.Wrapf(
				err,
				"cannot render ToComposite patches for composed resource %q",
				input.Spec.PatchTo))
			return rsp, nil
		}
	}

	if err = f.composed.ToResponse(rsp); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot convert composition to response %T", rsp))
		return
	}

	response.Normal(rsp, "Successful run")
	return rsp, nil
}

func (f *Function) subnetToCapaStruct(subnet *Subnet, region, providerConfig *string) (map[string]interface{}, error) {
	var (
		value  map[string]interface{}
		err    error
		public bool
	)

	if public, err = f.FindAWSPublicRouteTables(&subnet.ID, region, providerConfig); err != nil {
		return nil, err
	}

	subnet.IsPublic = public

	if err := composite.To(subnet, &value); err == nil {
		delete(value, "mapPublicIpOnLaunch")
	}

	return value, nil
}

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
