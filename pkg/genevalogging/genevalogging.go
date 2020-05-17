package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
)

type GenevaLogging interface {
	CreateOrUpdate(ctx context.Context) error
}

type genevaLogging struct {
	log *logrus.Entry

	resourceID               string
	acrName                  string
	namespace                string
	configVersion            string
	monitoringTenant         string
	monitoringGCSRegion      string
	monitoringGCSEnvironment string

	certs *v1.Secret

	cli    kubernetes.Interface
	seccli securityclient.Interface
}

func NewForOperator(log *logrus.Entry, cs *aro.ClusterSpec, cli kubernetes.Interface, seccli securityclient.Interface, certs *v1.Secret) GenevaLogging {
	return &genevaLogging{
		log: log,

		resourceID:               cs.ResourceID,
		acrName:                  cs.ACRName,
		namespace:                cs.GenevaLogging.Namespace,
		configVersion:            cs.GenevaLogging.ConfigVersion,
		monitoringGCSEnvironment: cs.GenevaLogging.MonitoringGCSEnvironment,
		monitoringGCSRegion:      cs.GenevaLogging.MonitoringGCSRegion,
		monitoringTenant:         cs.GenevaLogging.MonitoringTenant,

		certs: certs,

		cli:    cli,
		seccli: seccli,
	}
}

func (g *genevaLogging) fluentbitImage() string {
	return fmt.Sprintf(fluentbitImageFormat, g.acrName)
}

func (g *genevaLogging) mdsdImage() string {
	return fmt.Sprintf(mdsdImageFormat, g.acrName)
}

func (g *genevaLogging) ensureNamespace(ns string) error {
	_, err := g.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (g *genevaLogging) applyConfigMap(cm *v1.ConfigMap) error {
	_, err := g.cli.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_cm, err := g.cli.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		cm.ResourceVersion = _cm.ResourceVersion
		_, err = g.cli.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
		return err
	})
}

func (g *genevaLogging) applySecret(s *v1.Secret) error {
	_, err := g.cli.CoreV1().Secrets(s.Namespace).Create(s)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_s, err := g.cli.CoreV1().Secrets(s.Namespace).Get(s.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		s.ResourceVersion = _s.ResourceVersion
		_, err = g.cli.CoreV1().Secrets(s.Namespace).Update(s)
		return err
	})
}

func (g *genevaLogging) applyServiceAccount(sa *v1.ServiceAccount) error {
	_, err := g.cli.CoreV1().ServiceAccounts(sa.Namespace).Create(sa)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_sa, err := g.cli.CoreV1().ServiceAccounts(sa.Namespace).Get(sa.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		sa.ResourceVersion = _sa.ResourceVersion
		_, err = g.cli.CoreV1().ServiceAccounts(sa.Namespace).Update(sa)
		return err
	})
}

func (g *genevaLogging) applyDaemonSet(ds *appsv1.DaemonSet) error {
	_, err := g.cli.AppsV1().DaemonSets(ds.Namespace).Create(ds)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds, err := g.cli.AppsV1().DaemonSets(ds.Namespace).Get(ds.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		ds.ResourceVersion = _ds.ResourceVersion
		_, err = g.cli.AppsV1().DaemonSets(ds.Namespace).Update(ds)
		return err
	})
}

func (g *genevaLogging) CreateOrUpdate(ctx context.Context) error {
	r, err := azure.ParseResourceID(g.resourceID)
	if err != nil {
		return err
	}

	err = g.ensureNamespace(g.namespace)
	if err != nil {
		return err
	}

	err = g.applySecret(g.certs)
	if err != nil {
		return err
	}

	err = g.applyConfigMap(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-config",
			Namespace: g.namespace,
		},
		Data: map[string]string{
			"audit.conf":      auditConf,
			"containers.conf": containersConf,
			"journal.conf":    journalConf,
			"parsers.conf":    parsersConf,
		},
	})
	if err != nil {
		return err
	}

	err = g.applyServiceAccount(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "geneva",
			Namespace: g.namespace,
		},
	})
	if err != nil {
		return err
	}

	g.log.Print("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var scc *securityv1.SecurityContextConstraints
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		scc, err = g.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-genevalogging",
	}
	scc.Groups = nil
	scc.Users = []string{kubeServiceAccount}

	_, err = g.seccli.SecurityV1().SecurityContextConstraints().Create(scc)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return g.applyDaemonSet(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd",
			Namespace: g.namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mdsd"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "mdsd"},
					Annotations: map[string]string{"scheduler.alpha.kubernetes.io/critical-pod": ""},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "log",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "fluent",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/fluent",
								},
							},
						},
						{
							Name: "fluent-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "fluent-config",
									},
								},
							},
						},
						{
							Name: "machine-id",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/etc/machine-id",
								},
							},
						},
						{
							Name: "certificates",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "certificates",
								},
							},
						},
					},
					ServiceAccountName: "geneva",
					Tolerations: []v1.Toleration{
						{
							Effect:   v1.TaintEffectNoExecute,
							Operator: v1.TolerationOpExists,
						},
						{
							Effect:   v1.TaintEffectNoSchedule,
							Operator: v1.TolerationOpExists,
						},
					},
					Containers: []v1.Container{
						{
							Name:  "fluentbit-journal",
							Image: g.fluentbitImage(),
							Command: []string{
								"/opt/td-agent-bit/bin/td-agent-bit",
							},
							Args: []string{
								"-c",
								"/etc/td-agent-bit/journal.conf",
							},
							// TODO: specify requests/limits
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fluent-config",
									ReadOnly:  true,
									MountPath: "/etc/td-agent-bit",
								},
								{
									Name:      "machine-id",
									ReadOnly:  true,
									MountPath: "/etc/machine-id",
								},
								{
									Name:      "log",
									ReadOnly:  true,
									MountPath: "/var/log",
								},
								{
									Name:      "fluent",
									MountPath: "/var/lib/fluent",
								},
							},
						},
						{
							Name:  "fluentbit-containers",
							Image: g.fluentbitImage(),
							Command: []string{
								"/opt/td-agent-bit/bin/td-agent-bit",
							},
							Args: []string{
								"-c",
								"/etc/td-agent-bit/containers.conf",
							},
							// TODO: specify requests/limits
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fluent-config",
									ReadOnly:  true,
									MountPath: "/etc/td-agent-bit",
								},
								{
									Name:      "machine-id",
									ReadOnly:  true,
									MountPath: "/etc/machine-id",
								},
								{
									Name:      "log",
									ReadOnly:  true,
									MountPath: "/var/log",
								},
								{
									Name:      "fluent",
									MountPath: "/var/lib/fluent",
								},
							},
						},
						{
							Name:  "fluentbit-audit",
							Image: g.fluentbitImage(),
							Command: []string{
								"/opt/td-agent-bit/bin/td-agent-bit",
							},
							Args: []string{
								"-c",
								"/etc/td-agent-bit/audit.conf",
							},
							// TODO: specify requests/limits
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fluent-config",
									ReadOnly:  true,
									MountPath: "/etc/td-agent-bit",
								},
								{
									Name:      "machine-id",
									ReadOnly:  true,
									MountPath: "/etc/machine-id",
								},
								{
									Name:      "log",
									ReadOnly:  true,
									MountPath: "/var/log",
								},
								{
									Name:      "fluent",
									MountPath: "/var/lib/fluent",
								},
							},
						},
						{
							Name:  "mdsd",
							Image: g.mdsdImage(),
							Command: []string{
								"/usr/sbin/mdsd",
							},
							Args: []string{
								"-A",
								"-D",
								"-f",
								"24224",
								"-r",
								"/var/run/mdsd/default",
							},
							Env: []v1.EnvVar{
								{
									Name:  "MONITORING_GCS_ENVIRONMENT",
									Value: g.monitoringGCSEnvironment,
								},
								{
									Name:  "MONITORING_GCS_ACCOUNT",
									Value: "AROClusterLogs",
								},
								{
									Name:  "MONITORING_GCS_REGION",
									Value: g.monitoringGCSRegion,
								},
								{
									Name:  "MONITORING_GCS_CERT_CERTFILE",
									Value: "/etc/mdsd.d/secret/gcscert.pem",
								},
								{
									Name:  "MONITORING_GCS_CERT_KEYFILE",
									Value: "/etc/mdsd.d/secret/gcskey.pem",
								},
								{
									Name:  "MONITORING_GCS_NAMESPACE",
									Value: "AROClusterLogs",
								},
								{
									Name:  "MONITORING_CONFIG_VERSION",
									Value: g.configVersion,
								},
								{
									Name:  "MONITORING_USE_GENEVA_CONFIG_SERVICE",
									Value: "true",
								},
								{
									Name:  "MONITORING_TENANT",
									Value: g.monitoringTenant,
								},
								{
									Name:  "MONITORING_ROLE",
									Value: "cluster",
								},
								{
									Name: "MONITORING_ROLE_INSTANCE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "RESOURCE_ID",
									Value: strings.ToLower(g.resourceID),
								},
								{
									Name:  "SUBSCRIPTION_ID",
									Value: strings.ToLower(r.SubscriptionID),
								},
								{
									Name:  "RESOURCE_GROUP",
									Value: strings.ToLower(r.ResourceGroup),
								},
								{
									Name:  "RESOURCE_NAME",
									Value: strings.ToLower(r.ResourceName),
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("200m"),
									v1.ResourceMemory: resource.MustParse("1000Mi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("10m"),
									v1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "certificates",
									MountPath: "/etc/mdsd.d/secret",
								},
							},
						},
					},
				},
			},
		},
	})
}
