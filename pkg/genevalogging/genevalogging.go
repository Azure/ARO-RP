package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/tls"
)

const (
	fluentConf = `
	<source>
	@type systemd
	<storage>
	  @type local
	  path /var/log/journald.pos
	</storage>
	tag journald
  </source>
  <source>
	@type tail
	format json
	path /var/log/openshift-audit/*
	pos_file /var/log/openshift-audit.pos
	refresh_interval 10
	tag audit
	time_key timestamp
	time_format %Y-%m-%dT%H:%M:%SZ
  </source>
  <match logs>
	@type rewrite_tag_filter
	<rule>
	  key MESSAGE
	  pattern audit\.k8s\.io
	  tag audit
	</rule>
	<rule>
	  key MESSAGE
	  pattern .+
	  tag journald
	</rule>
  </match>
  <filter journald>
	@type record_transformer
	enable_ruby true
	<record>
	  MESSAGE ${record["MESSAGE"].nil? ? nil : record["MESSAGE"].force_encoding("UTF-8").encode("ASCII", invalid: :replace, undef: :replace)}
	  # data format:
	  # k8s_apiserver_apiserver-vqxg4_kube-service-catalog_72ce4d73-5224-11e9-98d0-000d3a196756_0
	  CONTAINER ${record["CONTAINER_NAME"].nil? ? nil : record["CONTAINER_NAME"].split("_")[1] }
	  POD ${record["CONTAINER_NAME"].nil? ? nil : record["CONTAINER_NAME"].split("_")[2] }
	  NAMESPACE ${record["CONTAINER_NAME"].nil? ? nil : record["CONTAINER_NAME"].split("_")[3] }
	  CONTAINER_ID ${record["CONTAINER_NAME"].nil? ? nil : record["CONTAINER_NAME"].split("_")[4] }
	</record>
  </filter>
  <match **>
	@type mdsd
	acktimeoutms 0
	buffer_type memory
	buffer_queue_full_action block
	disable_retry_limit true
	djsonsocket /var/run/mdsd/default_djson.socket
	emit_timestamp_name time
	flush_interval 10s
  </match>
`
	mdsdTemplateStr = `<?xml version="1.0" encoding="utf-8"?>
	<MonitoringManagement version="1.0" namespace="{{ .Namespace | XMLEscape }}" eventVersion="1" timestamp="2017-08-01T00:00:00.000Z">
		<Accounts>
			<Account moniker="{{ .AccountMoniker | XMLEscape }}" isDefault="true" autoKey="false"/>
		</Accounts>
		<Management eventVolume="Large" defaultRetentionInDays="90">
			<Identity tenantNameAlias="ResourceName">
				<IdentityComponent name="Region">{{ .Region | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="SubscriptionId">{{ .SubscriptionID | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="ResourceGroupName">{{ .ResourceGroupName | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="ResourceName">{{ .ResourceName | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="ResourceID">{{ .ResourceID | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="Role">{{ .Role | XMLEscape }}</IdentityComponent>
				<IdentityComponent name="RoleInstance" useComputerName="true"/>
			</Identity>
			<AgentResourceUsage diskQuotaInMB="50000"/>
		</Management>
		<Sources>
			<Source name="audit" dynamic_schema="true"/>
			<Source name="journald" dynamic_schema="true"/>
		</Sources>
		<Events>
			<MdsdEvents>
				<MdsdEventSource source="audit">
					<RouteEvent eventName="LinuxAsmAudit" storeType="CentralBond" priority="Normal"/>
				</MdsdEventSource>
				<MdsdEventSource source="journald">
					<RouteEvent eventName="CustomerSyslogEvents" storeType="CentralBond" priority="High"/>
				</MdsdEventSource>
			</MdsdEvents>
		</Events>
	</MonitoringManagement>
	`

	mainWrapper = `#!/bin/bash
echo "main wrapper"
mkdir -p /etc/mdsd.d/config
cp /geneva_config/mdsd.xml /etc/mdsd.d/config/

export GCS_AUTOMATIC_CONFIGURATION=0
export MDSD_COMPRESSION_ALGORITHM=lz4
export MDSD_COMPRESSION_LEVEL=4
unset MDSD_LOG_DIR

service cron start
/usr/sbin/mdsd -D -j -c /etc/mdsd.d/config/mdsd.xml
`
)

var (
	kubeNamespace      = "openshift-azure-logging"
	kubeServiceAccount = "system:serviceaccount:" + kubeNamespace + ":geneva"

	genevaNamespace      = "AROClusterLogs"
	genevaAccount        = genevaNamespace
	genevaAccountMoniker = strings.ToLower(genevaNamespace) + "diag"
	tdAgentImage         = "arosvc.azurecr.io/genevafluentd_td-agent:master_129"
	mdsdImage            = "arosvc.azurecr.io/genevamdsd:master_249"
	mdsdTemplate         = template.Must(template.New("").Funcs(map[string]interface{}{
		"XMLEscape": func(s string) (string, error) {
			var b bytes.Buffer
			err := xml.EscapeText(&b, []byte(s))
			return b.String(), err
		},
	}).Parse(mdsdTemplateStr))
)

type GenevaLogging interface {
	CreateOrUpdate(ctx context.Context) error
}

type genevaLogging struct {
	log *logrus.Entry

	env env.Interface
	oc  *api.OpenShiftCluster

	cli    kubernetes.Interface
	seccli securityclient.Interface
}

func New(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster, cli kubernetes.Interface, seccli securityclient.Interface) GenevaLogging {
	return &genevaLogging{
		log:    log,
		oc:     oc,
		env:    e,
		cli:    cli,
		seccli: seccli,
	}
}

func (g *genevaLogging) mdsdConfig() (string, error) {
	b := &bytes.Buffer{}
	resourceGroupName := g.oc.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(g.oc.Properties.ClusterProfile.ResourceGroupID, '/')+1:]
	err := mdsdTemplate.Execute(b, map[string]string{
		"Namespace":         genevaNamespace,
		"AccountMoniker":    genevaAccountMoniker,
		"Region":            g.env.Location(),
		"Role":              g.oc.Name,
		"SubscriptionID":    g.env.SubscriptionID(),
		"ResourceName":      g.oc.Name,
		"ResourceID":        strings.ToLower(g.oc.ID),
		"ResourceGroupName": resourceGroupName,
	})
	if err != nil {
		return "", err
	}
	return string(b.Bytes()), nil
}

func (g *genevaLogging) ensureNamespace() error {
	_, err := g.cli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeNamespace,
		},
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		res, err := g.cli.AuthorizationV1().SelfSubjectAccessReviews().Create(
			&authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: kubeNamespace,
						Verb:      "create",
						Resource:  "pods",
					},
				},
			},
		)
		if err != nil {
			return false, err
		}
		return res.Status.Allowed, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for self-sar: %v", err)
	}

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		sa, err := g.cli.CoreV1().ServiceAccounts(kubeNamespace).Get("default", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return len(sa.Secrets) > 0, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for default service account: %v", err)
	}

	err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		project, err := g.cli.CoreV1().Namespaces().Get(kubeNamespace, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		_, found := project.Annotations["openshift.io/sa.scc.uid-range"]
		return found, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for scc: %v", err)
	}

	return nil
}

func (g *genevaLogging) applyConfigMap(cm *v1.ConfigMap) error {
	_, err := g.cli.CoreV1().ConfigMaps(kubeNamespace).Create(cm)
	if err != nil && errors.IsAlreadyExists(err) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = g.cli.CoreV1().ConfigMaps(kubeNamespace).Update(cm)
			return err
		})
	}
	return err
}

func (g *genevaLogging) applySecret(s *v1.Secret) error {
	_, err := g.cli.CoreV1().Secrets(kubeNamespace).Create(s)
	if err != nil && errors.IsAlreadyExists(err) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = g.cli.CoreV1().Secrets(kubeNamespace).Update(s)
			return err
		})
	}
	return err
}

func (g *genevaLogging) applyServiceAccount(sa *v1.ServiceAccount) error {
	_, err := g.cli.CoreV1().ServiceAccounts(kubeNamespace).Create(sa)
	if err != nil && errors.IsAlreadyExists(err) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = g.cli.CoreV1().ServiceAccounts(kubeNamespace).Update(sa)
			return err
		})
	}
	return err
}

func (g *genevaLogging) applyDaemonSet(ds *appsv1.DaemonSet) error {
	_, err := g.cli.AppsV1().DaemonSets(kubeNamespace).Create(ds)
	if err != nil && errors.IsAlreadyExists(err) {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = g.cli.AppsV1().DaemonSets(kubeNamespace).Update(ds)
			return err
		})
	}
	return err
}

func retryOnNotFound(backoff wait.Backoff, fn func() error) error {
	return retry.OnError(backoff, errors.IsNotFound, fn)
}

func (g *genevaLogging) CreateOrUpdate(ctx context.Context) error {
	err := g.ensureNamespace()
	if err != nil {
		return err
	}

	err = g.applyConfigMap(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-config",
			Namespace: kubeNamespace,
		},
		Data: map[string]string{"fluent.conf": fluentConf},
	})
	if err != nil {
		return err
	}
	err = g.applyConfigMap(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd-wrapper",
			Namespace: kubeNamespace,
		},
		Data: map[string]string{"main-wrapper.sh": mainWrapper},
	})
	if err != nil {
		return err
	}

	mdsdConf, err := g.mdsdConfig()
	if err != nil {
		return err
	}
	err = g.applyConfigMap(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd-config",
			Namespace: kubeNamespace,
		},
		Data: map[string]string{"mdsd.xml": mdsdConf},
	})
	if err != nil {
		return err
	}
	key, certs, err := g.env.GenevaLoggingSecret()
	if err != nil {
		return err
	}
	gcsKeyBytes, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return err
	}
	gcsCertBytes, err := tls.CertAsBytes(certs[0])
	if err != nil {
		return err
	}

	err = g.applySecret(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcs-cert",
			Namespace: kubeNamespace,
		},
		StringData: map[string]string{
			"gcscert.pem": string(gcsCertBytes),
			"gcskey.pem":  string(gcsKeyBytes)},
	})
	if err != nil {
		return err
	}
	err = g.applyServiceAccount(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "geneva",
			Namespace: kubeNamespace,
		},
	})
	if err != nil {
		return err
	}

	err = retryOnNotFound(retry.DefaultRetry, func() error {
		_, err := g.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		return err
	})
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := g.seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
		if err != nil {
			return err
		}
		for _, user := range existing.Users {
			if user == kubeServiceAccount {
				g.log.Debugf("%s user found in privileged scc, no need to add", user)
				return nil
			}
		}
		existing.Users = append(existing.Users, kubeServiceAccount)
		_, err = g.seccli.SecurityV1().SecurityContextConstraints().Update(existing)
		return err
	})
	if err != nil {
		return err
	}
	return g.applyDaemonSet(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mdsd"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"scheduler.alpha.kubernetes.io/critical-pod": ""},
					Labels:      map[string]string{"app": "mdsd"},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "hostlog",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "mdsd-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "mdsd-config"},
								},
							},
						},
						{
							Name: "fluentd-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "fluentd-config"},
								},
							},
						},
						{
							Name: "mdsd-auth",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "gcs-cert",
								},
							},
						},
						{
							Name: "socket",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "mdsd-logs",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "mdsd-wrapper",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "mdsd-wrapper"},
									DefaultMode:          to.Int32Ptr(509),
								},
							},
						},
					},
					HostPID:            true,
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
							Name:  "td-agent",
							Image: tdAgentImage,
							Env: []v1.EnvVar{
								{
									Name:  "FLUENTD_CONF",
									Value: "/td-agent/config/fluent.conf",
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									/*"cpu":    resource.MustParse("500m"), this causes the pod to be unscheduleble on the compute nodes*/
									"memory": resource.MustParse("200Mi"),
								},
								Requests: v1.ResourceList{
									/*"cpu":    resource.MustParse("500m"), this causes the pod to be unscheduleble on the compute nodes*/
									"memory": resource.MustParse("200Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "socket",
									ReadOnly:  true,
									MountPath: "/var/run/mdsd/",
								},
								{
									Name:      "fluentd-config",
									MountPath: "/td-agent/config",
								},
								{
									Name:      "hostlog",
									MountPath: "/var/log/",
								},
							},
						},
						{
							Name:    "mdsd",
							Image:   mdsdImage,
							Command: []string{"/entrypoint/main-wrapper.sh"},
							Env: []v1.EnvVar{
								{
									Name:  "MDSD_CONTAINER_NAME",
									Value: "mdsd",
								},
								{
									Name:  "TENANT",
									Value: g.oc.Name,
								},
								{
									Name:  "MDSD_IMAGE",
									Value: mdsdImage,
								},
								{
									Name:  "MONITORING_GCS_ACCOUNT",
									Value: genevaAccount,
								},
								{
									Name:  "MONITORING_GCS_ENVIRONMENT",
									Value: g.env.GenevaLoggingEnvironment(),
								},
								{
									Name:  "MONITORING_GCS_REGION",
									Value: g.env.Location(),
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
									Name:  "MDSD_AUTH_DIR",
									Value: "/etc/mdsd.d/secret",
								},
								{
									Name:  "MDSD_CONFIG_DIR",
									Value: "/etc/mdsd.d/config",
								},
								{
									Name:  "MDSD_LOG_DIR",
									Value: "/var/log/mdsd",
								},
								{
									Name:  "MDSD_RUN_DIR",
									Value: "/var/run/mdsd",
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("400Mi"),
								},
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("400Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
								RunAsUser:  to.Int64Ptr(0),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "socket",
									MountPath: "/var/run/mdsd/",
								},
								{
									Name:      "mdsd-logs",
									MountPath: "/var/log/mdsd",
								},
								{
									Name:      "mdsd-auth",
									MountPath: "/etc/mdsd.d/secret",
								},
								{
									Name:      "mdsd-config",
									MountPath: "/geneva_config",
								},
								{
									Name:      "mdsd-wrapper",
									MountPath: "/entrypoint/main-wrapper.sh",
									SubPath:   "main-wrapper.sh",
								},
							},
						},
					},
				},
			},
		},
	})
}
