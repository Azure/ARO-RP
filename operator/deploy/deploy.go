package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/Azure/go-autorest/autorest/to"
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/genevalogging"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	KubeNamespace              = "openshift-azure-operator"
	kubeServiceAccountNameBase = "aro-operator-"
	kubeServiceAccountBase     = "system:serviceaccount:" + KubeNamespace + ":" + kubeServiceAccountNameBase
	aroOperatorImageFormat     = "%s/aro:%s"
)

type Operator interface {
	CreateOrUpdate(ctx context.Context) error
	IsReady() (bool, error)
}

type operator struct {
	log *logrus.Entry

	namespace         string
	imageVersion      string
	regTokens         map[string]string
	acrRegName        string
	genevaloggingKey  *rsa.PrivateKey
	genevaloggingCert *x509.Certificate
	servicePrincipal  []byte

	cluserSpec *aro.ClusterSpec

	dh     dynamichelper.DynamicHelper
	cli    kubernetes.Interface
	seccli securityclient.Interface
	arocli aroclient.AroV1alpha1Interface
}

func New(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster, cli kubernetes.Interface, seccli securityclient.Interface, arocli aroclient.AroV1alpha1Interface) (Operator, error) {
	restConfig, err := restconfig.RestConfig(e, oc)
	if err != nil {
		return nil, err
	}
	dh, err := dynamichelper.New(log, restConfig, dynamichelper.UpdatePolicy{
		IgnoreDefaults:                true,
		LogChanges:                    true,
		RetryOnConflict:               true,
		RefreshAPIResourcesOnNotFound: true,
	})
	if err != nil {
		return nil, err
	}

	key, cert := e.ClustersGenevaLoggingSecret()

	sp, err := json.Marshal(oc.Properties.ServicePrincipalProfile)
	if err != nil {
		return nil, err
	}

	o := &operator{
		log: log,

		namespace:         KubeNamespace,
		imageVersion:      version.GitCommit,
		acrRegName:        e.ACRName() + ".azurecr.io",
		regTokens:         map[string]string{},
		genevaloggingKey:  key,
		genevaloggingCert: cert,
		servicePrincipal:  sp,

		cluserSpec: &aro.ClusterSpec{
			ResourceID: oc.ID,
			GenevaLogging: aro.GenevaLoggingSpec{
				Namespace:                genevalogging.KubeNamespace,
				ACRName:                  e.ACRName(),
				ConfigVersion:            e.ClustersGenevaLoggingConfigVersion(),
				MonitoringGCSEnvironment: e.ClustersGenevaLoggingEnvironment(),
				MonitoringGCSRegion:      e.Location(),
				MonitoringTenant:         e.Location(),
			},
			ServicePrincipalValidation: aro.ServicePrincipalValidationSpec{
				MasterSubnetID: oc.Properties.MasterProfile.SubnetID,
			},
		},

		dh:     dh,
		cli:    cli,
		seccli: seccli,
		arocli: arocli,
	}
	for _, wp := range oc.Properties.WorkerProfiles {
		o.cluserSpec.ServicePrincipalValidation.WorkerSubnetIDs = append(o.cluserSpec.ServicePrincipalValidation.WorkerSubnetIDs, wp.SubnetID)
	}

	for _, reg := range oc.Properties.RegistryProfiles {
		if reg.Name == o.acrRegName && string(reg.Password) != "" {
			o.regTokens[o.acrRegName] = reg.Username + ":" + string(reg.Password)
		}
	}
	if _, ok := e.(env.Dev); ok {
		auths, err := pullsecret.Auths([]byte(os.Getenv("PULL_SECRET")))
		if err != nil {
			return nil, err
		}
		o.regTokens[o.acrRegName] = auths[o.acrRegName]["auth"].(string)
	}
	return o, nil
}

func (o *operator) aroOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}
	return fmt.Sprintf(aroOperatorImageFormat, o.acrRegName, o.imageVersion)
}

func (o *operator) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	scc, err := o.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	scc.TypeMeta = metav1.TypeMeta{
		Kind:       "SecurityContextConstraints",
		APIVersion: "security.openshift.io/v1",
	}
	scc.ObjectMeta = metav1.ObjectMeta{
		Name: name,
	}
	scc.Groups = []string{}
	scc.Users = []string{serviceAccountName}
	return scc, nil
}

func (o *operator) deployment(role string) *appsv1.Deployment {
	deploymentName := "aro-operator-" + role
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: o.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: to.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploymentName},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": deploymentName},
				},
				Spec: v1.PodSpec{
					PriorityClassName:  "system-cluster-critical",
					ServiceAccountName: kubeServiceAccountNameBase + role,
					NodeSelector:       map[string]string{"node-role.kubernetes.io/worker": ""},
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
							Env: []v1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if role == "master" {
		deployment.Spec.Template.Spec.Volumes = []v1.Volume{
			{
				Name: "pullsecret-tokens",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "pullsecret-tokens",
					},
				},
			},
		}
		deployment.Spec.Template.Spec.NodeSelector = map[string]string{"node-role.kubernetes.io/master": ""}
		deployment.Spec.Template.Spec.Tolerations = []v1.Toleration{
			{
				Operator: v1.TolerationOpExists,
				Key:      "node-role.kubernetes.io/master",
				Effect:   v1.TaintEffectNoSchedule,
			},
		}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{
				Name:      "pullsecret-tokens",
				ReadOnly:  true,
				MountPath: "/pull-secrets",
			},
		}
		deployment.Spec.Template.Spec.Containers[0].SecurityContext = &v1.SecurityContext{
			Privileged: to.BoolPtr(true),
			RunAsUser:  to.Int64Ptr(0),
		}
	}
	return deployment
}

func (o *operator) resources(ctx context.Context) ([]runtime.Object, error) {
	// first static resources from Assets
	results := []runtime.Object{}
	for _, assetName := range AssetNames() {
		b, err := Asset(assetName)
		if err != nil {
			return nil, err
		}
		obj := &unstructured.Unstructured{}
		err = yaml.Unmarshal(b, obj)
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

	ssc, err := o.securityContextConstraints(ctx, "privileged-aro-operator", kubeServiceAccountBase+"master")
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
			Type:       corev1.SecretTypeOpaque,
			StringData: o.regTokens,
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service-principal",
				Namespace: o.namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"servicePrincipal": o.servicePrincipal},
		},
		ssc,
		o.deployment("master"),
		o.deployment("worker"),
		&aro.Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: "aro.openshift.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: *o.cluserSpec,
		},
	} {
		results = append(results, obj)
	}

	return results, nil
}

var kindOrder = map[string]int{
	"CustomResourceDefinition":   1,
	"Namespace":                  2,
	"ClusterRole":                3,
	"ClusterRoleBinding":         4,
	"SecurityContextConstraints": 5,
	"ServiceAccount":             6,
	"Secret":                     7,
	"Deployment":                 8,
	"Cluster":                    9,
}

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	resources, err := o.resources(ctx)
	if err != nil {
		return err
	}

	objects := []*unstructured.Unstructured{}
	for _, res := range resources {
		un, err := dynamichelper.ToUnstructured(res)
		if err != nil {
			return err
		}
		objects = append(objects, un)
	}

	dynamichelper.HashWorkloadConfigs(objects)

	sort.Slice(objects, func(i, j int) bool {
		return kindOrder[objects[i].GetKind()] < kindOrder[objects[j].GetKind()]
	})
	for _, un := range objects {
		err = o.dh.CreateOrUpdate(ctx, un)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *operator) IsReady() (bool, error) {
	ok, err := ready.CheckDeploymentIsReady(o.cli.AppsV1().Deployments(o.namespace), "aro-operator-master")()
	if !ok || err != nil {
		return ok, err
	}
	return ready.CheckDeploymentIsReady(o.cli.AppsV1().Deployments(o.namespace), "aro-operator-worker")()
}
