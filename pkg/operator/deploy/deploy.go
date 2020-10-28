package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Operator interface {
	CreateOrUpdate(context.Context) error
	IsReady(context.Context) (bool, error)
}

type operator struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	dh     dynamichelper.DynamicHelper
	cli    kubernetes.Interface
	extcli extensionsclient.Interface
	arocli aroclient.AroV1alpha1Interface
}

func New(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, cli kubernetes.Interface, extcli extensionsclient.Interface, arocli aroclient.AroV1alpha1Interface) (Operator, error) {
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
		arocli: arocli,
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
			if d.Labels == nil {
				d.Labels = map[string]string{}
			}
			d.Labels["version"] = version.GitCommit
			for i := range d.Spec.Template.Spec.Containers {
				d.Spec.Template.Spec.Containers[i].Image = o.env.AROOperatorImage()

				if o.env.DeploymentMode() == deployment.Development {
					d.Spec.Template.Spec.Containers[i].Env = append(d.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
						Name:  "RP_MODE",
						Value: "development",
					})
				}
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgoperator.SecretName,
				Namespace: pkgoperator.Namespace,
			},
			Data: map[string][]byte{
				genevalogging.GenevaCertName: gcsCertBytes,
				genevalogging.GenevaKeyName:  gcsKeyBytes,
				v1.DockerConfigJsonKey:       []byte(ps),
			},
		},
		&arov1alpha1.Cluster{
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

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	resources, err := o.resources()
	if err != nil {
		return err
	}

	uns := make([]*unstructured.Unstructured, 0, len(resources))
	for _, res := range resources {
		un := &unstructured.Unstructured{}
		err = scheme.Scheme.Convert(res, un, nil)
		if err != nil {
			return err
		}
		uns = append(uns, un)
	}

	sort.Slice(uns, func(i, j int) bool {
		return dynamichelper.CreateOrder(uns[i], uns[j])
	})

	for _, un := range uns {
		err = o.dh.Ensure(ctx, un)
		if err != nil {
			return err
		}

		switch un.GroupVersionKind().GroupKind().String() {
		case "CustomResourceDefinition.apiextensions.k8s.io":
			err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
				crd, err := o.extcli.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, un.GetName(), metav1.GetOptions{})
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

		case "Cluster.aro.openshift.io":
			// add an owner reference onto our configuration secret.  This is
			// can only be done once we've got the cluster UID.  It is needed to
			// ensure that secret updates trigger updates of the appropriate
			// controllers
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				cluster, err := o.arocli.Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				s, err := o.cli.CoreV1().Secrets(pkgoperator.Namespace).Get(ctx, pkgoperator.SecretName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				err = controllerutil.SetControllerReference(cluster, s, scheme.Scheme)
				if err != nil {
					return err
				}

				_, err = o.cli.CoreV1().Secrets(pkgoperator.Namespace).Update(ctx, s, metav1.UpdateOptions{})
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *operator) IsReady(ctx context.Context) (bool, error) {
	ok, err := ready.CheckDeploymentIsReady(ctx, o.cli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master")()
	if !ok || err != nil {
		return ok, err
	}
	ok, err = ready.CheckDeploymentIsReady(ctx, o.cli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker")()
	if !ok || err != nil {
		return ok, err
	}

	// wait for conditions to appear
	cluster, err := o.arocli.Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, ct := range arov1alpha1.AllConditionTypes() {
		cond := cluster.Status.Conditions.GetCondition(ct)
		if cond == nil {
			return false, nil
		}
		if cond.Status != corev1.ConditionTrue {
			return false, nil
		}
	}
	return true, nil
}

func isCRDEstablished(crd *extv1beta1.CustomResourceDefinition) bool {
	m := make(map[extv1beta1.CustomResourceDefinitionConditionType]extv1beta1.ConditionStatus, len(crd.Status.Conditions))
	for _, cond := range crd.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[extv1beta1.Established] == extv1beta1.ConditionTrue &&
		m[extv1beta1.NamesAccepted] == extv1beta1.ConditionTrue
}
