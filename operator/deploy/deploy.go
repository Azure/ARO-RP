package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/genevalogging"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	KubeNamespace          = "openshift-azure-operator"
	kubeServiceAccountName = "aro-operator"
	kubeServiceAccount     = "system:serviceaccount:" + KubeNamespace + ":" + kubeServiceAccountName
	aroOperatorImageFormat = "%s/aro:%s"
)

type Operator interface {
	CreateOrUpdate(ctx context.Context) error
}

type operator struct {
	log *logrus.Entry

	resourceID        string
	namespace         string
	imageVersion      string
	acrToken          string
	acrRegName        string
	acrName           string
	genevaloggingKey  *rsa.PrivateKey
	genevaloggingCert *x509.Certificate

	genevaloggingSpec *aro.GenevaLoggingSpec

	restconfig *rest.Config
	cli        kubernetes.Interface
	seccli     securityclient.Interface
	arocli     aroclient.AroV1alpha1Interface
}

func New(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster, cli kubernetes.Interface, seccli securityclient.Interface, arocli aroclient.AroV1alpha1Interface) (Operator, error) {
	var acrToken string
	acrRegName := e.ACRName() + ".azurecr.io"
	for i, rp := range oc.Properties.RegistryProfiles {
		if rp.Name == acrRegName {
			acrToken = oc.Properties.RegistryProfiles[i].Username + ":" + string(oc.Properties.RegistryProfiles[i].Password)
		}
	}
	restConfig, err := restconfig.RestConfig(e, oc)
	if err != nil {
		return nil, err
	}

	key, cert := e.ClustersGenevaLoggingSecret()

	return &operator{
		log: log,

		resourceID:        oc.ID,
		namespace:         KubeNamespace,
		imageVersion:      version.GitCommit,
		acrName:           e.ACRName(),
		acrToken:          acrToken,
		acrRegName:        acrRegName,
		genevaloggingKey:  key,
		genevaloggingCert: cert,

		genevaloggingSpec: &aro.GenevaLoggingSpec{
			Namespace:                genevalogging.KubeNamespace,
			ConfigVersion:            e.ClustersGenevaLoggingConfigVersion(),
			MonitoringGCSEnvironment: e.ClustersGenevaLoggingEnvironment(),
			MonitoringGCSRegion:      e.Location(),
			MonitoringTenant:         e.Location(),
		},

		restconfig: restConfig,
		cli:        cli,
		seccli:     seccli,
		arocli:     arocli,
	}, nil
}

func (o *operator) aroOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}
	return fmt.Sprintf(aroOperatorImageFormat, o.acrRegName, o.imageVersion)
}

func (o *operator) applyClusterCR(cluster *aro.Cluster) error {
	_, err := o.arocli.Clusters().Create(cluster)
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_c, err := o.arocli.Clusters().Get(cluster.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		cluster.ResourceVersion = _c.ResourceVersion
		_, err = o.arocli.Clusters().Update(cluster)
		return err
	})
}

func (o *operator) applySecret(s *corev1.Secret) error {
	_, err := o.cli.CoreV1().Secrets(s.Namespace).Create(s)
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_s, err := o.cli.CoreV1().Secrets(s.Namespace).Get(s.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		s.ResourceVersion = _s.ResourceVersion
		_, err = o.cli.CoreV1().Secrets(s.Namespace).Update(s)
		return err
	})
}

func (o *operator) applyDeployment(ds *appsv1.Deployment) error {
	_, err := o.cli.AppsV1().Deployments(ds.Namespace).Create(ds)
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := o.cli.AppsV1().Deployments(ds.Namespace).Get(ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = o.cli.AppsV1().Deployments(ds.Namespace).Update(ds)
		return err
	})
}

func (o *operator) applyAssets(ctx context.Context) error {
	dh, err := dynamichelper.New(o.log, o.restconfig, true, true)
	if err != nil {
		return err
	}

	b, err := Asset("resources.yaml")
	if err != nil {
		return err
	}

	manifests := strings.Split(string(b), "---")
	for _, manifeststr := range manifests {
		obj, err := dh.UnmarshalYAML([]byte(manifeststr))
		if err != nil {
			return err
		}
		o.log.Infof("applyAsset %s %s %s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
		err = dh.CreateOrUpdate(ctx, &obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	err := o.applyAssets(ctx)
	if err != nil {
		return err
	}

	// create a secret here for genevalogging, later we will copy it to
	// the genevalogging namespace.
	gcsKeyBytes, err := tls.PrivateKeyAsBytes(o.genevaloggingKey)
	if err != nil {
		return err
	}

	gcsCertBytes, err := tls.CertAsBytes(o.genevaloggingCert)
	if err != nil {
		return err
	}

	err = o.applySecret(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "certificates",
			Namespace: o.namespace,
		},
		StringData: map[string]string{
			"gcscert.pem": string(gcsCertBytes),
			"gcskey.pem":  string(gcsKeyBytes),
		},
	})
	if err != nil {
		return err
	}

	err = o.applySecret(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pullsecret-tokens",
			Namespace: o.namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			o.acrRegName: o.acrToken,
		},
	})
	if err != nil {
		return err
	}

	o.log.Print("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var scc *securityv1.SecurityContextConstraints
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		scc, err = o.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-operator",
	}
	scc.Groups = nil
	scc.Users = []string{kubeServiceAccount}

	_, err = o.seccli.SecurityV1().SecurityContextConstraints().Create(scc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	err = o.applyDeployment(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-operator",
			Namespace: o.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "aro"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "aro"},
				},
				Spec: v1.PodSpec{
					PriorityClassName: "system-cluster-critical",
					Volumes: []v1.Volume{
						{
							Name: "pullsecret-tokens",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "pullsecret-tokens",
								},
							},
						},
					},
					ServiceAccountName: kubeServiceAccountName,
					Containers: []v1.Container{
						{
							Name:  "aro-operator",
							Image: o.aroOperatorImage(),
							Command: []string{
								"aro",
							},
							Args: []string{
								"operator",
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "pullsecret-tokens",
									ReadOnly:  true,
									MountPath: "/pull-secrets",
								},
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return err
	}

	o.log.Print("waiting for operator to come up")
	timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		dep, err := o.cli.AppsV1().Deployments(o.namespace).Get("aro-operator", metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		return dep.Status.AvailableReplicas == 1, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	return o.applyClusterCR(&aro.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: aro.ClusterSpec{
			ResourceID:    o.resourceID,
			ACRName:       o.acrName,
			GenevaLogging: *o.genevaloggingSpec,
		},
	})
}
