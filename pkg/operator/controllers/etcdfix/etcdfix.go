package etcdfix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"

	"github.com/Azure/go-autorest/autorest/to"
	projectv1 "github.com/openshift/api/project/v1"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	serviceAccountName  = "etcd-fix"
	kubeName            = "etcd-fix"
	kubeNamespace       = "openshift-etcd"
	kubeServiceAccount  = "system:serviceaccount" + kubeNamespace + ":" + serviceAccountName
	configMapName       = "etcd-fix-master-ip-change"
	configMapScriptName = "etcd-fix.sh"
	configMapScriptDir  = "/tmp"
	image               = "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c09189b0b03bd631ad04db7f0519d717fec4a11321d2d665ee9dc10c11466f7c"

	jsonPath = `'{ .spec.containers[0].env[4] } { .spec.containers[0].env[-8, -5, -2] } { .status.conditions[2].status }'`
)

//go:embed etcd-fix-operator.sh
var shellScriptEtcdFix string

func (r *Reconciler) resources(ctx context.Context, cluster *arov1alpha1.Cluster) ([]kruntime.Object, error) {
	scc, err := r.securityContextConstraints(ctx, "privileged-etcdfix", kubeServiceAccount)
	if err != nil {
		return nil, err
	}
	resourceCPU, err := resource.ParseQuantity("10m")
	if err != nil {
		return nil, err
	}
	resourceMemory, err := resource.ParseQuantity("300Mi")
	if err != nil {
		return nil, err
	}

	return []kruntime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        kubeNamespace,
				Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
			},
		},
		scc,
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: kubeNamespace,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceAccountName,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"operator.openshift.io"},
					Resources: []string{"etcds.operator.openshift.io", "pods"},
					Verbs:     []string{"get", "list", "patch"},
				},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubeName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: kubeNamespace,
				},
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     "system:cluster-admins",
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: kubeNamespace,
			},
			Data: map[string]string{
				configMapScriptName: shellScriptEtcdFix,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubeName,
				Namespace: kubeNamespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": kubeName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": kubeName},
					},
					Spec: corev1.PodSpec{
						NodeSelector:       map[string]string{"node-role.kubernetes.io/master": ""},
						HostPID:            true,
						ServiceAccountName: serviceAccountName,
						Containers: []corev1.Container{
							{
								Name:    "etcd-fix-master-node-ip-change",
								Image:   image,
								Command: []string{"/host/usr/bin/oc"},
								Args: []string{
									"observe",
									"pod",
									"-n",
									"openshift-etcd",
									"-l",
									"app=etcd",
									"--template",
									jsonPath,
									"--type-env-var=EVENT",
									"--",
									configMapScriptDir + "/" + configMapScriptName,
								},
								Ports: []corev1.ContainerPort{
									{
										HostPort:      11256,
										ContainerPort: 11251,
									},
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"/usr/bin/bash",
												"-c",
												"echo etcd-fix stopping",
											},
										},
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resourceCPU,
										corev1.ResourceMemory: resourceMemory,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name: "K8S_NODE",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "spec.nodeName",
											},
										},
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "host",
										MountPath: "/host",
										ReadOnly:  false,
									},
									{
										Name:      configMapName,
										MountPath: configMapScriptDir + "/" + configMapScriptName,
										SubPath:   configMapScriptName,
										ReadOnly:  true,
									},
								},
							},
						},
						Tolerations: []corev1.Toleration{
							{
								Effect:   corev1.TaintEffectNoExecute,
								Operator: corev1.TolerationOpExists,
							},
							{
								Key: "node-role.kubernetes.io/master",
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
							{
								Name: configMapName,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: configMapName,
										},
										DefaultMode: to.Int32Ptr(0555),
									},
								},
							},
						},
						DNSPolicy:     corev1.DNSClusterFirst,
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
		},
	}, nil
}

func (r *Reconciler) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	scc, err := r.securitycli.SecurityV1().SecurityContextConstraints().Get(ctx, "privileged", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	scc.ObjectMeta = metav1.ObjectMeta{
		Name: name,
	}
	scc.Groups = []string{}
	scc.Users = []string{serviceAccountName}
	return scc, nil
}
