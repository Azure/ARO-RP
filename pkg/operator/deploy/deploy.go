package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
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

	arocli        aroclient.Interface
	extensionscli extensionsclient.Interface
	kubernetescli kubernetes.Interface
	dh            dynamichelper.Interface
}

func New(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, arocli aroclient.Interface, extensionscli extensionsclient.Interface, kubernetescli kubernetes.Interface) (Operator, error) {
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

		arocli:        arocli,
		extensionscli: extensionscli,
		kubernetescli: kubernetescli,
		dh:            dh,
	}, nil
}

func (o *operator) resources() ([]kruntime.Object, error) {
	// first static resources from Assets
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
			if d.Labels == nil {
				d.Labels = map[string]string{}
			}
			d.Labels["version"] = version.GitCommit
			for i := range d.Spec.Template.Spec.Containers {
				d.Spec.Template.Spec.Containers[i].Image = o.env.AROOperatorImage()

				if o.env.IsLocalDevelopmentMode() {
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
	key, cert := o.env.ClusterGenevaLoggingSecret()
	gcsKeyBytes, err := utiltls.PrivateKeyAsBytes(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := utiltls.CertAsBytes(cert)
	if err != nil {
		return nil, err
	}

	ps, err := pullsecret.Build(o.oc, "")
	if err != nil {
		return nil, err
	}

	vnetID, _, err := subnet.Split(o.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	domain := o.oc.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + o.env.Domain()
	}

	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:             o.oc.ID,
			ClusterResourceGroupID: o.oc.Properties.ClusterProfile.ResourceGroupID,
			Domain:                 domain,
			ACRDomain:              o.env.ACRDomain(),
			AZEnvironment:          o.env.Environment().Name,
			Location:               o.env.Location(),
			InfraID:                o.oc.Properties.InfraID,
			ArchitectureVersion:    int(o.oc.Properties.ArchitectureVersion),
			VnetID:                 vnetID,
			GenevaLogging: arov1alpha1.GenevaLoggingSpec{
				ConfigVersion:            o.env.ClusterGenevaLoggingConfigVersion(),
				MonitoringGCSAccount:     o.env.ClusterGenevaLoggingAccount(),
				MonitoringGCSEnvironment: o.env.ClusterGenevaLoggingEnvironment(),
				MonitoringGCSNamespace:   o.env.ClusterGenevaLoggingNamespace(),
			},
			InternetChecker: arov1alpha1.InternetCheckerSpec{
				URLs: []string{
					fmt.Sprintf("https://%s/", o.env.ACRDomain()),
					o.env.Environment().ActiveDirectoryEndpoint,
					o.env.Environment().ResourceManagerEndpoint,
					o.env.Environment().GenevaMonitoringEndpoint,
				},
			},

			APIIntIP:                 o.oc.Properties.APIServerProfile.IntIP,
			IngressIP:                o.oc.Properties.IngressProfiles[0].IP,
			GatewayPrivateEndpointIP: o.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP,
			// Update the OperatorFlags from the version in the RP
			OperatorFlags: arov1alpha1.OperatorFlags(o.oc.Properties.OperatorFlags),
		},
	}

	// TODO (BV): reenable gateway once we fix bugs
	// if o.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
	// 	cluster.Spec.GatewayDomains = append(o.env.GatewayDomains(), o.oc.Properties.ImageRegistryStorageAccountName+".blob."+o.env.Environment().StorageEndpointSuffix)
	// }

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
				corev1.DockerConfigJsonKey:   []byte(ps),
			},
		},
		cluster,
	), nil
}

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	resources, err := o.resources()
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		err = o.dh.Ensure(ctx, resource)
		if err != nil {
			return err
		}

		gvks, _, err := scheme.Scheme.ObjectKinds(resource)
		if err != nil {
			return err
		}

		switch gvks[0].GroupKind().String() {
		case "CustomResourceDefinition.apiextensions.k8s.io":
			acc, err := meta.Accessor(resource)
			if err != nil {
				return err
			}

			err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
				crd, err := o.extensionscli.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, acc.GetName(), metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				return isCRDEstablished(crd), nil
			})
			if err != nil {
				return err
			}

			err = o.dh.Refresh()
			if err != nil {
				return err
			}

		case "Cluster.aro.openshift.io":
			// add an owner reference onto our configuration secret.  This is
			// can only be done once we've got the cluster UID.  It is needed to
			// ensure that secret updates trigger updates of the appropriate
			// controllers
			err = retry.OnError(wait.Backoff{
				Steps:    60,
				Duration: time.Second,
			}, func(err error) bool {
				// IsForbidden here is intended to catch the following transient
				// error: secrets "cluster" is forbidden: cannot set
				// blockOwnerDeletion in this case because cannot find
				// RESTMapping for APIVersion aro.openshift.io/v1alpha1 Kind
				// Cluster: no matches for kind "Cluster" in version
				// "aro.openshift.io/v1alpha1"
				return kerrors.IsForbidden(err) || kerrors.IsConflict(err)
			}, func() error {
				cluster, err := o.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				s, err := o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Get(ctx, pkgoperator.SecretName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				err = controllerutil.SetControllerReference(cluster, s, scheme.Scheme)
				if err != nil {
					return err
				}

				_, err = o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Update(ctx, s, metav1.UpdateOptions{})
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
	ok, err := ready.CheckDeploymentIsReady(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master")()
	if !ok || err != nil {
		return ok, err
	}
	ok, err = ready.CheckDeploymentIsReady(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker")()
	if !ok || err != nil {
		return ok, err
	}

	return true, nil
}

func isCRDEstablished(crd *extensionsv1.CustomResourceDefinition) bool {
	m := make(map[extensionsv1.CustomResourceDefinitionConditionType]extensionsv1.ConditionStatus, len(crd.Status.Conditions))
	for _, cond := range crd.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[extensionsv1.Established] == extensionsv1.ConditionTrue &&
		m[extensionsv1.NamesAccepted] == extensionsv1.ConditionTrue
}
