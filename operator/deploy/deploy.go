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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/genevalogging"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/jsonpath"
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

	dh     dynamichelper.DynamicHelper
	cli    kubernetes.Interface
	seccli securityclient.Interface
	arocli aroclient.AroV1alpha1Interface
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
	dh, err := dynamichelper.New(log, restConfig, dynamichelper.UpdatePolicy{
		IgnoreDefaults:  true,
		LogChanges:      true,
		RetryOnConflict: true,
	})
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

		dh:     dh,
		cli:    cli,
		seccli: seccli,
		arocli: arocli,
	}, nil
}

func (o *operator) aroOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}
	return fmt.Sprintf(aroOperatorImageFormat, o.acrRegName, o.imageVersion)
}

func (o *operator) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	o.log.Print("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var scc *securityv1.SecurityContextConstraints
	var err error
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		scc, err = o.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return nil, err
	}

	scc.TypeMeta = metav1.TypeMeta{
		Kind:       "SecurityContextConstraints",
		APIVersion: "security.openshift.io/v1",
	}
	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-operator",
	}
	scc.Groups = []string{}
	scc.Users = []string{kubeServiceAccount}
	return scc, nil
}

func (o *operator) deployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aro-operator",
			Namespace: o.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: to.Int32Ptr(1),
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
	}
}

func clean(o unstructured.Unstructured) {
	gk := o.GroupVersionKind().GroupKind()

	jsonpath.MustCompile("$.status").Delete(o.Object)
	jsonpath.MustCompile("$.metadata.creationTimestamp").Delete(o.Object)

	switch gk.String() {
	case "Deployment.apps":
		jsonpath.MustCompile("$.spec.template.metadata.creationTimestamp").Delete(o.Object)
	}
}

func (o *operator) apply(ctx context.Context, ro runtime.Object) error {
	b, err := yaml.Marshal(ro)
	if err != nil {
		return err
	}
	obj := &unstructured.Unstructured{}
	err = yaml.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	clean(*obj)

	o.log.Infof("applyAsset %s %s %s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
	return o.dh.CreateOrUpdate(ctx, obj)
}

func (o *operator) resources(ctx context.Context) ([]runtime.Object, error) {
	// first static resources from Assets
	b, err := Asset("resources.yaml")
	if err != nil {
		return nil, err
	}
	results := []runtime.Object{}
	manifests := strings.Split(string(b), "---")
	for _, manifeststr := range manifests {
		obj := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(manifeststr), obj)
		if err != nil {
			return nil, err
		}
		results = append(results, obj)
	}

	// then dynamic resources
	gcsKeyBytes, err := tls.PrivateKeyAsBytes(o.genevaloggingKey)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := tls.CertAsBytes(o.genevaloggingCert)
	if err != nil {
		return nil, err
	}

	ssc, err := o.securityContextConstraints(ctx, "privileged-operator", kubeServiceAccount)
	if err != nil {
		return nil, err
	}

	// create a secret here for genevalogging, later we will copy it to
	// the genevalogging namespace.

	for _, obj := range []runtime.Object{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "certificates",
				Namespace: o.namespace,
			},
			StringData: map[string]string{
				"gcscert.pem": string(gcsCertBytes),
				"gcskey.pem":  string(gcsKeyBytes),
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pullsecret-tokens",
				Namespace: o.namespace,
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				o.acrRegName: o.acrToken,
			},
		},
		ssc,
		o.deployment(),
		&aro.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "aro.openshift.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: aro.ClusterSpec{
				ResourceID:    o.resourceID,
				ACRName:       o.acrName,
				GenevaLogging: *o.genevaloggingSpec,
			},
		},
	} {
		results = append(results, obj)
	}

	return results, nil
}

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	resources, err := o.resources(ctx)
	if err != nil {
		return err
	}
	for _, res := range resources {
		err = o.apply(ctx, res)
		if err != nil {
			return err
		}
	}

	o.log.Print("waiting for operator to come up")
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		dep, err := o.cli.AppsV1().Deployments(o.namespace).Get("aro-operator", metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		return dep.Status.AvailableReplicas == 1, nil
	}, timeoutCtx.Done())
}
