package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/genevalogging"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

const (
	ACRPullSecretName = "acr-pullsecret-tokens"
)

type Operator interface {
	CreateOrUpdate() error
	IsReady() (bool, error)
}

type operator struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	dh     dynamichelper.DynamicHelper
	cli    kubernetes.Interface
	extcli extensionsclient.Interface
}

func New(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, cli kubernetes.Interface, extcli extensionsclient.Interface) (Operator, error) {
	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return nil, err
	}

	return &operator{
		log: log,
		env: env,
		oc:  oc,

		dh:     dh,
		cli:    cli,
		extcli: extcli,
	}, nil
}

func (o *operator) resources() ([]runtime.Object, error) {
	// first static resources from Assets
	results := []runtime.Object{}
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
				d.Spec.Template.Spec.Containers[i].Image = o.env.AROOperatorImage()
			}
		}

		results = append(results, obj)
	}
	// then dynamic resources
	key, cert := o.env.ClustersGenevaLoggingSecret()
	gcsKeyBytes, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := tls.CertAsBytes(cert)
	if err != nil {
		return nil, err
	}

	ps, err := pullsecret.Build(o.oc, "")
	if err != nil {
		return nil, err
	}

	// create a secret here for genevalogging, later we will copy it to
	// the genevalogging namespace.
	return append(results,
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      genevalogging.CertificatesSecretName,
				Namespace: pkgoperator.Namespace,
			},
			Data: map[string][]byte{
				"gcscert.pem": gcsCertBytes,
				"gcskey.pem":  gcsKeyBytes,
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ACRPullSecretName,
				Namespace: pkgoperator.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{v1.DockerConfigJsonKey: []byte(ps)},
		},
		&arov1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "aro.openshift.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
			Spec: arov1alpha1.ClusterSpec{
				ResourceID: o.oc.ID,
				ACRName:    o.env.ACRName(),
				Location:   o.env.Location(),
				GenevaLogging: arov1alpha1.GenevaLoggingSpec{
					ConfigVersion:            o.env.ClustersGenevaLoggingConfigVersion(),
					MonitoringGCSEnvironment: o.env.ClustersGenevaLoggingEnvironment(),
				},
				InternetChecker: arov1alpha1.InternetCheckerSpec{
					URLs: []string{
						"https://arosvc.azurecr.io/",
						"https://login.microsoftonline.com/",
						"https://management.azure.com/",
						"https://gcs.prod.monitoring.core.windows.net/",
					},
				},
			},
		},
	), nil
}

func (o *operator) CreateOrUpdate() error {
	resources, err := o.resources()
	if err != nil {
		return err
	}

	objects := []*unstructured.Unstructured{}
	for _, res := range resources {
		un := &unstructured.Unstructured{}
		err = scheme.Scheme.Convert(res, un, nil)
		if err != nil {
			return err
		}
		objects = append(objects, un)
	}

	sort.Slice(objects, func(i, j int) bool {
		return dynamichelper.KindLess(objects[i].GetKind(), objects[j].GetKind())
	})
	for _, un := range objects {
		err = o.dh.Ensure(un)
		if err != nil {
			return err
		}

		if un.GroupVersionKind().GroupKind().String() == "CustomResourceDefinition.apiextensions.k8s.io" {
			err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
				crd, err := o.extcli.ApiextensionsV1beta1().CustomResourceDefinitions().Get(un.GetName(), metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				return isCRDEstablished(crd), nil
			})
			if err != nil {
				return err
			}

			err = o.dh.RefreshAPIResources()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *operator) IsReady() (bool, error) {
	ok, err := ready.CheckDeploymentIsReady(o.cli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master")()
	if !ok || err != nil {
		return ok, err
	}
	return ready.CheckDeploymentIsReady(o.cli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker")()
}

func isCRDEstablished(crd *extv1beta1.CustomResourceDefinition) bool {
	m := make(map[extv1beta1.CustomResourceDefinitionConditionType]extv1beta1.ConditionStatus, len(crd.Status.Conditions))
	for _, cond := range crd.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[extv1beta1.Established] == extv1beta1.ConditionTrue &&
		m[extv1beta1.NamesAccepted] == extv1beta1.ConditionTrue
}
