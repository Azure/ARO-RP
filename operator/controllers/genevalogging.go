package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aro "github.com/Azure/ARO-RP/operator/api/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

const (
	fluentbitImageFormat = "%s.azurecr.io/fluentbit:1.3.9-1"
	mdsdImageFormat      = "%s.azurecr.io/genevamdsd:master_279"

	parsersConf = `
[PARSER]
	Name audit
	Format json
	Time_Key stageTimestamp
	Time_Format %Y-%m-%dT%H:%M:%S.%L

[PARSER]
	Name containerpath
	Format regex
	Regex ^/var/log/containers/(?<POD>[^_]+)_(?<NAMESPACE>[^_]+)_(?<CONTAINER>.+)-(?<CONTAINER_ID>[0-9a-f]{64})\.log$

[PARSER]
	Name crio
	Format regex
	Regex ^(?<TIMESTAMP>[^ ]+) [^ ]+ [^ ]+ (?<MESSAGE>.*)$
	Time_Key TIMESTAMP
	Time_Format %Y-%m-%dT%H:%M:%S.%L
`

	journalConf = `
[INPUT]
	Name systemd
	Tag journald
	DB /var/lib/fluent/journald

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP
	Remove SYSLOG_FACILITY

[OUTPUT]
	Name forward
	Port 24224
`

	containersConf = `
[SERVICE]
	Parsers_File /etc/td-agent-bit/parsers.conf

[INPUT]
	Name tail
	Path /var/log/containers/*
	Path_Key path
	Tag containers
	DB /var/lib/fluent/containers
	Parser crio

[FILTER]
	Name parser
	Match containers
	Key_Name path
	Parser containerpath
	Reserve_Data true

[FILTER]
	Name grep
	Match containers
	Regex NAMESPACE ^(?:default|kube-.*|openshift|openshift-.*)$

[OUTPUT]
	Name forward
	Port 24224
`

	auditConf = `
[SERVICE]
	Parsers_File /etc/td-agent-bit/parsers.conf

[INPUT]
	Name tail
	Path /var/log/kube-apiserver/audit*
	Path_Key path
	Tag audit
	DB /var/lib/fluent/audit
	Parser audit

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under user
	Add_prefix user_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under impersonatedUser
	Add_prefix impersonatedUser_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under responseStatus
	Add_prefix responseStatus_

[FILTER]
	Name nest
	Match *
	Operation lift
	Nested_under objectRef
	Add_prefix objectRef_

[OUTPUT]
	Name forward
	Port 24224
`
)

func kubeServiceAccount(namespace string) string {
	return "system:serviceaccount:" + namespace + ":geneva"
}

func (g *GenevaloggingReconciler) fluentbitImage(acrName string) string {
	return fmt.Sprintf(fluentbitImageFormat, acrName)
}

func (g *GenevaloggingReconciler) mdsdImage(acrName string) string {
	return fmt.Sprintf(mdsdImageFormat, acrName)
}

func (g *GenevaloggingReconciler) ensureNamespace(ctx context.Context, cs *aro.Cluster) error {
	err := g.Client.Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cs.Spec.GenevaLogging.Namespace,
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (g *GenevaloggingReconciler) applyConfigMap(ctx context.Context, cs *aro.Cluster, cm *v1.ConfigMap) error {
	err := g.Client.Create(ctx, cm)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_cm := &v1.ConfigMap{}
		err = g.Client.Get(ctx, client.ObjectKey{Namespace: cm.Namespace, Name: cm.Name}, _cm)
		if err != nil {
			return err
		}
		g.Log.Info("applyConfigMap")
		cm.ResourceVersion = _cm.ResourceVersion
		return g.Client.Update(ctx, cm)
	})
}

func (g *GenevaloggingReconciler) applySecret(ctx context.Context, cs *aro.Cluster, s *v1.Secret) error {
	err := g.Client.Create(ctx, s)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_s := &v1.Secret{}
		err = g.Client.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: s.Name}, _s)
		if err != nil {
			return err
		}
		g.Log.Info("applySecret")

		s.ResourceVersion = _s.ResourceVersion
		return g.Client.Update(ctx, s)
	})
}

func (g *GenevaloggingReconciler) applyServiceAccount(ctx context.Context, cs *aro.Cluster, sa *v1.ServiceAccount) error {
	err := g.Client.Create(ctx, sa)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_sa := &v1.ServiceAccount{}
		err = g.Client.Get(ctx, client.ObjectKey{Namespace: sa.Namespace, Name: sa.Name}, _sa)
		if err != nil {
			return err
		}
		g.Log.Info("applyServiceAccount")

		sa.ResourceVersion = _sa.ResourceVersion
		return g.Client.Update(ctx, sa)
	})
}

func (g *GenevaloggingReconciler) applyDaemonSet(ctx context.Context, cs *aro.Cluster, ds *appsv1.DaemonSet) error {
	err := g.Client.Create(ctx, ds)
	if !errors.IsAlreadyExists(err) {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ds := &appsv1.DaemonSet{}
		if err = g.Client.Get(ctx, client.ObjectKey{Namespace: ds.Namespace, Name: ds.Name}, _ds); err != nil {
			return err
		}
		g.Log.Info("applyDaemonSet")

		ds.ResourceVersion = _ds.ResourceVersion
		return g.Client.Update(ctx, ds)
	})
}

func (g *GenevaloggingReconciler) createSecurityContextConstraint(ctx context.Context, cs *aro.Cluster) error {
	g.Log.Info("waiting for privileged security context constraint")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	scc := &securityv1.SecurityContextConstraints{}
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		err := g.Client.Get(ctx, client.ObjectKey{Name: "privileged"}, scc)
		g.Log.Info(" scc privileged", "err", err)
		return err == nil, nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	scc.ObjectMeta = metav1.ObjectMeta{
		Name: "privileged-genevalogging",
	}
	scc.Groups = nil
	scc.Users = []string{kubeServiceAccount(cs.Spec.GenevaLogging.Namespace)}

	err = g.Client.Create(ctx, scc)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (g *GenevaloggingReconciler) clustersGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate) {
	// TODO
	return nil, nil
}

func (g *GenevaloggingReconciler) applyCertificates(ctx context.Context, cs *aro.Cluster) error {
	key, cert := g.clustersGenevaLoggingSecret()

	gcsKeyBytes, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return err
	}

	gcsCertBytes, err := tls.CertAsBytes(cert)
	if err != nil {
		return err
	}
	return g.applySecret(ctx, cs, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "certificates",
			Namespace: cs.Spec.GenevaLogging.Namespace,
		},
		StringData: map[string]string{
			"gcscert.pem": string(gcsCertBytes),
			"gcskey.pem":  string(gcsKeyBytes),
		},
	})
}

func (g *GenevaloggingReconciler) reconsileGenevaLogging(ctx context.Context, cs *aro.Cluster) error {
	r, err := azure.ParseResourceID(cs.Spec.ResourceID)
	if err != nil {
		return err
	}
	err = g.ensureNamespace(ctx, cs)
	if err != nil {
		return err
	}

	err = g.applyConfigMap(ctx, cs, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-config",
			Namespace: cs.Spec.GenevaLogging.Namespace,
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

	/* TODO
	err = g.applyCertificates(ctx, cs)
	if err != nil {
		return err
	}
	*/
	err = g.applyServiceAccount(ctx, cs, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "geneva",
			Namespace: cs.Spec.GenevaLogging.Namespace,
		},
	})
	if err != nil {
		return err
	}

	/* TODO the scc client is not registered
	err = g.createSecurityContextConstraint(ctx, cs)
	if err != nil {
		return err
	}
	*/
	return g.applyDaemonSet(ctx, cs, &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd",
			Namespace: cs.Spec.GenevaLogging.Namespace,
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
							Image: g.fluentbitImage(cs.Spec.ACRName),
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
							Image: g.fluentbitImage(cs.Spec.ACRName),
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
							Image: g.fluentbitImage(cs.Spec.ACRName),
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
							Image: g.mdsdImage(cs.Spec.ACRName),
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
									Value: cs.Spec.GenevaLogging.MonitoringGCSEnvironment,
								},
								{
									Name:  "MONITORING_GCS_ACCOUNT",
									Value: "AROClusterLogs",
								},
								{
									Name:  "MONITORING_GCS_REGION",
									Value: cs.Spec.GenevaLogging.MonitoringGCSRegion,
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
									Value: cs.Spec.GenevaLogging.ConfigVersion,
								},
								{
									Name:  "MONITORING_USE_GENEVA_CONFIG_SERVICE",
									Value: "true",
								},
								{
									Name:  "MONITORING_TENANT",
									Value: cs.Spec.GenevaLogging.MonitoringTenant,
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
									Value: strings.ToLower(cs.Spec.ResourceID),
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
