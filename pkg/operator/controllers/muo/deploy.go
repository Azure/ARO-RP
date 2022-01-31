package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

type Deployer interface {
	CreateOrUpdate(context.Context, *arov1alpha1.Cluster) error
	Remove(context.Context) error
	IsReady(ctx context.Context) (bool, error)
	Resources(string) ([]kruntime.Object, error)
}

type deployer struct {
	kubernetescli kubernetes.Interface
	dh            dynamichelper.Interface
}

func newDeployer(kubernetescli kubernetes.Interface, dh dynamichelper.Interface) Deployer {
	return &deployer{
		kubernetescli: kubernetescli,
		dh:            dh,
	}
}

func (o *deployer) Resources(pullspec string) ([]kruntime.Object, error) {
	results := []kruntime.Object{}
	for _, assetName := range AssetNames() {
		b, err := Asset(assetName)
		if err != nil {
			return nil, err
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, nil, nil)
		if err != nil {
			return nil, err
		}

		// set the image for the deployments
		if d, ok := obj.(*appsv1.Deployment); ok {
			for i := range d.Spec.Template.Spec.Containers {
				d.Spec.Template.Spec.Containers[i].Image = pullspec
			}
		}

		results = append(results, obj)
	}

	return results, nil
}

func (o *deployer) CreateOrUpdate(ctx context.Context, cluster *arov1alpha1.Cluster) error {
	imagePullspec, ext := cluster.Spec.OperatorFlags[controllerPullSpec]
	if !ext {
		return errors.New("missing pullspec")
	}

	resources, err := o.Resources(imagePullspec)
	if err != nil {
		return err
	}

	err = dynamichelper.SetControllerReferences(resources, cluster)
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	return o.dh.Ensure(ctx, resources...)
}

func (o *deployer) Remove(ctx context.Context) error {
	resources, err := o.Resources("")
	if err != nil {
		return err
	}

	var errs []error
	for _, obj := range resources {
		// delete any deployments we have
		if d, ok := obj.(*appsv1.Deployment); ok {
			err := o.dh.EnsureDeleted(ctx, "Deployment", d.Namespace, d.Name)
			// Don't error out because then we might delete some resources and not others
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) != 0 {
		errContent := []string{"error removing MUO:"}
		for _, err := range errs {
			errContent = append(errContent, err.Error())
		}
		return errors.New(strings.Join(errContent, "\n"))
	}

	return nil
}

func (o *deployer) IsReady(ctx context.Context) (bool, error) {
	return ready.CheckDeploymentIsReady(ctx, o.kubernetescli.AppsV1().Deployments("openshift-managed-upgrade-operator"), "managed-upgrade-operator")()
}
