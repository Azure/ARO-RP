package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ugorji/go/codec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

type muoConfig struct {
	api.MissingFields
	ConfigManager struct {
		api.MissingFields
		Source          string `json:"source,omitempty"`
		OcmBaseUrl      string `json:"ocmBaseUrl,omitempty"`
		LocalConfigName string `json:"localConfigName,omitempty"`
	} `json:"configManager,omitempty"`
}

type Deployer interface {
	CreateOrUpdate(context.Context, *arov1alpha1.Cluster, *config.MUODeploymentConfig) error
	Remove(context.Context) error
	IsReady(ctx context.Context) (bool, error)
	Resources(*config.MUODeploymentConfig) ([]kruntime.Object, error)
}

type deployer struct {
	kubernetescli kubernetes.Interface
	dh            dynamichelper.Interface

	jsonHandle *codec.JsonHandle
}

func newDeployer(kubernetescli kubernetes.Interface, dh dynamichelper.Interface) Deployer {
	return &deployer{
		kubernetescli: kubernetescli,
		dh:            dh,

		jsonHandle: new(codec.JsonHandle),
	}
}

func (o *deployer) Resources(config *config.MUODeploymentConfig) ([]kruntime.Object, error) {
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
				d.Spec.Template.Spec.Containers[i].Image = config.Pullspec
			}
		}

		if cm, ok := obj.(*corev1.ConfigMap); ok {
			if cm.Name == "managed-upgrade-operator-config" && cm.Namespace == "openshift-managed-upgrade-operator" {
				// read the config.yaml from the MUO ConfigMap which stores defaults
				configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
				if err != nil {
					return nil, err
				}

				var configData muoConfig
				err = codec.NewDecoderBytes(configDataJSON, o.jsonHandle).Decode(&configData)
				if err != nil {
					return nil, err
				}

				if config.EnableConnected {
					configData.ConfigManager.Source = "OCM"
					configData.ConfigManager.OcmBaseUrl = config.OCMBaseURL
					configData.ConfigManager.LocalConfigName = ""
				} else {
					configData.ConfigManager.Source = "LOCAL"
					configData.ConfigManager.LocalConfigName = "managed-upgrade-config"
					configData.ConfigManager.OcmBaseUrl = ""
				}

				// Write the yaml back into the ConfigMap
				var b []byte
				err = codec.NewEncoderBytes(&b, o.jsonHandle).Encode(configData)
				if err != nil {
					return nil, err
				}

				cmYaml, err := yaml.JSONToYAML(b)
				if err != nil {
					return nil, err
				}
				cm.Data["config.yaml"] = string(cmYaml)
			}
		}

		results = append(results, obj)
	}

	return results, nil
}

func (o *deployer) CreateOrUpdate(ctx context.Context, cluster *arov1alpha1.Cluster, config *config.MUODeploymentConfig) error {
	resources, err := o.Resources(config)
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
	resources, err := o.Resources(&config.MUODeploymentConfig{})
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
