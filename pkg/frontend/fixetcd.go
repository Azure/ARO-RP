package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	securityv1client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	serviceAccountName                  = "etcd-recovery-privileged"
	kubeServiceAccount                  = "system:serviceaccount" + nameSpaceEtcds + ":" + serviceAccountName
	nameSpaceEtcds                      = "openshift-etcd"
	image                               = "ubi8/ubi-minimal"
	jobName                             = "etcd-recovery-"
	jobNameDataBackup                   = jobName + "data-backup"
	jobNameFixPeers                     = jobName + "fix-peers"
	patchOverrides                      = "unsupportedConfigOverrides:"
	patchDisableOverrides               = `{"useUnsupportedUnsafeNonHANonProductionUnstableEtcd": true}`
	ctxKey                timeoutAdjust = "TESTING"
)

type timeoutAdjust string

type degradedEtcd struct {
	Node  string
	Pod   string
	NewIP string
	OldIP string
}

func (m *manager) fixEtcd(ctx context.Context) error {
	pods, err := m.kubernetescli.CoreV1().Pods(nameSpaceEtcds).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	// Exit early if degradedEtcd is empty, an error is returned
	// no further actions should be taken
	de, err := comparePodEnvToIP(m.log, pods)
	if err != nil {
		return err
	}
	m.log.Infof("Found degraded endpoint: %v", de)

	err = backupEtcdData(ctx, m.log, de.Node, m.kubernetescli.BatchV1())
	if err != nil {
		return err
	}

	err = fixPeers(ctx, m.log, de, pods, m.kubernetescli, m.securitycli.SecurityV1().SecurityContextConstraints(), m.doc.OpenShiftCluster.Name)
	if err != nil {
		return err
	}

	etcd, err := m.operatorcli.OperatorV1().Etcds().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	existingOverrides := etcd.Spec.UnsupportedConfigOverrides.Raw
	etcd.Spec.UnsupportedConfigOverrides = kruntime.RawExtension{
		Raw: []byte(patchDisableOverrides),
	}
	err = patchEtcd(ctx, m.log, m.operatorcli.OperatorV1().Etcds(), etcd, patchDisableOverrides)
	if err != nil {
		return err
	}

	err = deleteSecrets(ctx, m.log, m.kubernetescli.CoreV1().Secrets(nameSpaceEtcds), de)
	if err != nil {
		return err
	}

	etcd.Spec.ForceRedeploymentReason = fmt.Sprintf("single-master-recovery-%s", time.Now())
	err = patchEtcd(ctx, m.log, m.operatorcli.OperatorV1().Etcds(), etcd, etcd.Spec.ForceRedeploymentReason)
	if err != nil {
		return err
	}

	etcd.Spec.OperatorSpec.UnsupportedConfigOverrides.Raw = existingOverrides
	return patchEtcd(ctx, m.log, m.operatorcli.OperatorV1().Etcds(), etcd, patchOverrides+etcd.Spec.OperatorSpec.UnsupportedConfigOverrides.String())
}

func patchEtcd(ctx context.Context, log *logrus.Entry, operatercli operatorv1client.EtcdInterface, e *operatorv1.Etcd, patch string) error {
	e.CreationTimestamp = metav1.Time{
		Time: time.Now(),
	}
	e.ResourceVersion = ""
	e.SelfLink = ""
	e.UID = ""

	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(e)
	if err != nil {
		return err
	}

	_, err = operatercli.Patch(ctx, e.Name, types.MergePatchType, buf.Bytes(), metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Infof("Patched etcd %s with %s", e.Name, patch)

	return nil
}

func deleteSecrets(ctx context.Context, log *logrus.Entry, secretcli corev1client.SecretInterface, de *degradedEtcd) error {
	for _, prefix := range []string{"etcd-peer-", "etcd-serving-", "etcd-serving-metrics-"} {
		secret := prefix + de.Node
		log.Infof("Deleting secret %s", secret)
		err := secretcli.Delete(ctx, secret, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func getPeerPods(pods []corev1.Pod, de *degradedEtcd, cluster string) (string, error) {
	regNode, err := regexp.Compile(".master-[0-9]$")
	if err != nil {
		return "", err
	}
	regPod, err := regexp.Compile("etcd-" + cluster + "-[0-9A-Za-z]*-master-[0-9]$")
	if err != nil {
		return "", err
	}

	var peerPods string
	for _, p := range pods {
		if regNode.MatchString(p.Spec.NodeName) && regPod.MatchString(p.Name) && p.Name != de.Pod {
			peerPods += p.Name + " "
		}
	}
	return peerPods, nil
}

func fixPeers(ctx context.Context, log *logrus.Entry, de *degradedEtcd, pods *corev1.PodList, kubecli kubernetes.Interface, securitycli securityv1client.SecurityContextConstraintsInterface, cluster string) error {
	peerPods, err := getPeerPods(pods.Items, de, cluster)
	if err != nil {
		return err
	}

	cleanup, err := createPrivilegedServiceAccount(ctx, log, serviceAccountName, kubeServiceAccount, kubecli, securitycli)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Infof("Creating job %s", jobNameFixPeers)
	b, err := kubecli.BatchV1().Jobs(nameSpaceEtcds).Create(ctx, &batchv1.Job{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobNameFixPeers,
			Namespace: nameSpaceEtcds,
			Labels:    map[string]string{"app": jobNameDataBackup},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobNameFixPeers,
					Namespace: nameSpaceEtcds,
					Labels:    map[string]string{"app": jobNameDataBackup},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "backup",
							Image: image,
							Command: []string{
								"/bin/bash",
								"-cx",
								backupOrFixEtcd,
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: to.BoolPtr(true),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PEER_PODS",
									Value: peerPods,
								},
								{
									Name:  "DEGRADED_NODE",
									Value: de.Node,
								},
								{
									Name:  "FIX_PEERS",
									Value: "true",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "host",
									MountPath: "/host",
									ReadOnly:  false,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "host",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{
		FieldManager: jobNameFixPeers,
	})
	if err != nil {
		return err
	}

	timeout := adjustTimeout(ctx, log)
	log.Infof("Waiting for %s", jobNameFixPeers)
	select {
	case <-time.After(timeout):
		log.Infof("Finished waiting for job %s", jobNameFixPeers)
		break
	case <-ctx.Done():
		log.Warnf("Request cancelled while waiting for %s", jobNameFixPeers)
	}

	log.Infof("Deleting %s now", jobNameFixPeers)
	propPolicy := metav1.DeletePropagationBackground
	err = kubecli.BatchV1().Jobs(b.Namespace).Delete(ctx, jobNameFixPeers, metav1.DeleteOptions{
		PropagationPolicy: &propPolicy,
	})
	if err != nil {
		return err
	}

	// return errors from deferred cleanup function
	return err
}

func createPrivilegedServiceAccount(ctx context.Context, log *logrus.Entry, name, usersAccount string, kubecli kubernetes.Interface, securitycli securityv1client.SecurityContextConstraintsInterface) (func() error, error) {
	log.Infof("Creating service account %s now", name)
	serviceAcc, err := kubecli.CoreV1().ServiceAccounts(nameSpaceEtcds).Create(ctx, &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		AutomountServiceAccountToken: to.BoolPtr(true),
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	log.Infof("Creating cluster role %s now", usersAccount)
	cr, err := kubecli.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: usersAccount,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "create"},
				Resources: []string{"pods", "pods/exec"},
				APIGroups: []string{""},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	log.Infof("Creating cluster role binding %s now", serviceAccountName)
	crb, err := kubecli.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     kubeServiceAccount,
			APIGroup: "",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: nameSpaceEtcds,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	log.Infof("Creating Security Context Constraint %s now", name)
	scc, err := securitycli.Create(ctx, &securityv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpaceEtcds,
		},
		Groups:                   []string{},
		Users:                    []string{usersAccount},
		AllowPrivilegedContainer: true,
		AllowPrivilegeEscalation: to.BoolPtr(true),
		AllowedCapabilities:      []corev1.Capability{"*"},
		RunAsUser: securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyRunAsAny,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return func() error {
		log.Infof("Deleting service account %s now", serviceAcc.Name)
		err := kubecli.CoreV1().ServiceAccounts(nameSpaceEtcds).Delete(ctx, serviceAcc.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		log.Infof("Deleting security context contstraint %s now", scc.Name)
		err = securitycli.Delete(ctx, scc.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		log.Infof("Deleting cluster role %s now", cr.Name)
		err = kubecli.RbacV1().ClusterRoles().Delete(ctx, cr.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		log.Infof("Deleting cluster role binding %s now", crb.Name)
		err = kubecli.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		return nil
	}, nil
}

func backupEtcdData(ctx context.Context, log *logrus.Entry, node string, batch batchv1client.BatchV1Interface) error {
	log.Infof("Creating job %s", jobNameDataBackup)
	job, err := batch.Jobs(nameSpaceEtcds).Create(ctx, &batchv1.Job{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobNameDataBackup,
			Namespace: nameSpaceEtcds,
			Labels:    map[string]string{"app": jobNameDataBackup},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   jobNameDataBackup,
					Labels: map[string]string{"app": jobNameDataBackup},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					NodeName:      node,
					Containers: []corev1.Container{
						{
							Name:  "backup",
							Image: image,
							Command: []string{
								"chroot",
								"/host",
								"/bin/bash",
								"-c",
								backupOrFixEtcd,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "host",
									MountPath: "/host",
									ReadOnly:  false,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_CHROOT"},
								},
								Privileged: to.BoolPtr(true),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "BACKUP",
									Value: "true",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "host",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{
		FieldManager: jobNameDataBackup,
	})
	if err != nil {
		return err
	}

	timeout := adjustTimeout(ctx, log)
	log.Infof("Waiting for %s", jobNameDataBackup)
	select {
	case <-time.After(timeout):
		log.Infof("Finished waiting for job %s", jobNameDataBackup)
	case <-ctx.Done():
		log.Warnf("Request cancelled while waiting for %s", jobNameDataBackup)
	}

	log.Infof("Deleting job %s now", jobNameDataBackup)
	propPolicy := metav1.DeletePropagationBackground
	return batch.Jobs(job.Namespace).Delete(ctx, jobNameDataBackup, metav1.DeleteOptions{
		PropagationPolicy: &propPolicy,
	})
}

func adjustTimeout(ctx context.Context, log *logrus.Entry) time.Duration {
	if ctx.Value(ctxKey) == "TRUE" {
		log.Info("Timeout adjusted for testing environment")
		return time.Microsecond
	}

	return time.Minute
}

func comparePodEnvToIP(log *logrus.Entry, pods *corev1.PodList) (*degradedEtcd, error) {
	for _, p := range pods.Items {
		etcdIP := ipFromEnv(p.Spec.Containers, p.Name)
		for _, ip := range p.Status.PodIPs {
			if ip.IP != etcdIP && etcdIP != "" {
				log.Infof("Found conflicting IPs for etcd Pod %s: %s!=%s", p.Name, ip.IP, etcdIP)
				return &degradedEtcd{
					Node:  strings.ReplaceAll(p.Name, "etcd-", ""),
					Pod:   p.Name,
					NewIP: ip.IP,
					OldIP: etcdIP,
				}, nil
			}
		}
	}
	return &degradedEtcd{}, errors.New("degradedEtcd is empty, unable to remediate etcd deployment")
}

func ipFromEnv(containers []corev1.Container, podName string) string {
	for _, c := range containers {
		if c.Name == "etcd" {
			for _, e := range c.Env {
				envName := strings.ReplaceAll(strings.ReplaceAll(podName, "-", "_"), "etcd_", "NODE_")
				if e.Name == fmt.Sprintf("%s_IP", envName) {
					return e.Value
				}
			}
		}
	}
	return ""
}
