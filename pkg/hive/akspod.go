package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

type aksInstallationManager struct {
	log *logrus.Entry
	env env.Core

	kubernetescli kubernetes.Interface

	dh dynamichelper.Interface
}

const clusterInstallerAccount = "aro-installer"
const installerJobName = "aro-installer"

// NewAKSManagerFromHiveManager creates an AKS installation manager from the
// Hive ClusterManager
func NewAKSManagerFromHiveManager(h ClusterManager) (*aksInstallationManager, error) {
	m, ok := h.(*clusterManager)
	if !ok {
		return nil, errors.New("not a Hive clustermanager?")
	}

	return &aksInstallationManager{
		log: m.log,
		env: m.env,

		kubernetescli: m.kubernetescli,

		dh: m.dh,
	}, nil
}

func (c *aksInstallationManager) clustermanagerrole(doc *api.OpenShiftClusterDocument) []kruntime.Object {
	return []kruntime.Object{
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterInstallerAccount,
				Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterInstallerAccount + "-role",
				Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets", "configmaps"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterInstallerAccount + "-binding",
				Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      clusterInstallerAccount,
					Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Name: clusterInstallerAccount + "-role",
				Kind: "Role",
			},
		},
	}
}

func (c *aksInstallationManager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error {
	sppSecret, err := servicePrincipalSecretForInstall(doc.OpenShiftCluster, sub, c.env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	psSecret, err := pullsecretSecret(doc.OpenShiftCluster.Properties.HiveProfile.Namespace, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	resources := []kruntime.Object{
		sppSecret,
		psSecret,
		c.installPod(sub, doc, version),
	}

	resources = append(resources, c.clustermanagerrole(doc)...)

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	err = c.dh.Ensure(ctx, resources...)
	if err != nil {
		return err
	}

	return nil
}

func (c *aksInstallationManager) installPod(sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) *batchv1.Job {
	var envVars = []corev1.EnvVar{
		{
			Name:  "ARO_UUID",
			Value: doc.ID,
		},
		{
			Name:  "OPENSHIFT_INSTALL_INVOKER",
			Value: "hive",
		},
		{
			Name:  "OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE",
			Value: version.Properties.OpenShiftPullspec,
		},
	}

	if c.env.IsLocalDevelopmentMode() {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "ARO_RP_MODE",
			Value: "development",
		})
		for _, i := range devEnvVars {
			envVars = append(envVars, makeEnvSecret(i))
		}
	} else {
		for _, i := range prodEnvVars {
			envVars = append(envVars, makeEnvSecret(i))
		}
	}

	provisionJobDeadline := time.Hour

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installerJobName,
			Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
		},

		Spec: batchv1.JobSpec{
			BackoffLimit:          pointer.Int32Ptr(0),
			Completions:           pointer.Int32Ptr(1),
			ActiveDeadlineSeconds: pointer.Int64Ptr(int64(provisionJobDeadline.Seconds())),

			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: clusterInstallerAccount,
					DNSPolicy:          corev1.DNSClusterFirst,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Image: version.Properties.InstallerPullspec,
							Env:   envVars,
							Command: []string{
								"/bin/bash",
								"-c",
								"/bin/openshift-install create manifests && /bin/openshift-install create cluster",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "spp",
									MountPath: "/.azure",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "spp",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: clusterServicePrincipalSecretName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *aksInstallationManager) IsClusterInstallationComplete(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	job, err := c.kubernetescli.BatchV1().Jobs(doc.OpenShiftCluster.Properties.HiveProfile.Namespace).Get(ctx, installerJobName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	checkFailureConditions := map[batchv1.JobConditionType]corev1.ConditionStatus{
		batchv1.JobFailed: corev1.ConditionTrue,
	}

	checkSuccessConditions := map[batchv1.JobConditionType]corev1.ConditionStatus{
		batchv1.JobComplete: corev1.ConditionTrue,
	}

	for _, cond := range job.Status.Conditions {
		conditionStatus, found := checkFailureConditions[cond.Type]
		if found && conditionStatus == cond.Status {
			return false, fmt.Errorf("clusterdeployment has failed: %s == %s", cond.Type, cond.Status)
		}

		conditionStatus, found = checkSuccessConditions[cond.Type]
		if found && conditionStatus == cond.Status {
			return true, nil
		}
	}

	return false, nil
}
