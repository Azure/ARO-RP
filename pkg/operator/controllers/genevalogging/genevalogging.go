package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	projectv1 "github.com/openshift/api/project/v1"
	securityv1 "github.com/openshift/api/security/v1"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var privilegedNamespaceLabels = map[string]string{
	"pod-security.kubernetes.io/enforce": "privileged",
	"pod-security.kubernetes.io/audit":   "privileged",
	"pod-security.kubernetes.io/warn":    "privileged",
}

const (
	masterRoleLabel       = "node-role.kubernetes.io/master"
	controlPlaneRoleLabel = "node-role.kubernetes.io/control-plane"
)

func (r *Reconciler) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	scc := &securityv1.SecurityContextConstraints{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "privileged"}, scc)
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

// namespaceLabels adds proper namespace labels for the privileged geneva logging
// daemonset on OpenShift 4.11+
func (r *Reconciler) namespaceLabels(ctx context.Context) (map[string]string, error) {
	usePodSecurityAdmission, err := pkgoperator.ShouldUsePodSecurityStandard(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	if usePodSecurityAdmission {
		return privilegedNamespaceLabels, nil
	}

	return map[string]string{}, nil
}

func (r *Reconciler) resources(ctx context.Context, cluster *arov1alpha1.Cluster) ([]kruntime.Object, error) {
	scc, err := r.securityContextConstraints(ctx, "privileged-genevalogging", kubeServiceAccount)
	if err != nil {
		return nil, err
	}

	nsLabels, err := r.namespaceLabels(ctx)
	if err != nil {
		return nil, err
	}

	resources := []kruntime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        kubeNamespace,
				Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
				Labels:      nsLabels,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "geneva",
				Namespace: kubeNamespace,
			},
		},
		scc,
	}

	profiles, err := getOTelProfiles(cluster.Spec.OperatorFlags)
	if err != nil {
		return nil, err
	}

	resources = append(resources,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      otelConfigMapName,
				Namespace: kubeNamespace,
			},
			Data: map[string]string{
				// config.yaml remains for compatibility with existing tooling/tests.
				"config.yaml":       selectOTelConfig(profiles.master),
				otelMasterConfigKey: selectOTelConfig(profiles.master),
				otelWorkerConfigKey: selectOTelConfig(profiles.worker),
			},
		},
	)

	gatewayTarget, targetReady, err := telemetryGatewayTarget(cluster)
	if err != nil {
		return nil, err
	}

	// During early install, the gateway endpoint may not be populated yet.
	// Create config resources now and defer daemonset creation until ready.
	if !targetReady {
		return resources, nil
	}

	daemonsets, err := r.otelDaemonSets(cluster, gatewayTarget.endpoint, gatewayTarget.hostAliases)
	if err != nil {
		return nil, err
	}
	for _, ds := range daemonsets {
		resources = append(resources, ds)
	}

	return resources, nil
}

func selectOTelConfig(profile otelProfile) string {
	cfg, err := renderOTelConfig(profile)
	if err != nil {
		cfg, reducedErr := renderOTelConfig(otelProfileReducedLogs)
		if reducedErr != nil {
			return ""
		}
		return cfg
	}
	return cfg
}

type telemetryGatewayTargetSpec struct {
	endpoint    string
	hostAliases []corev1.HostAlias
}

func telemetryGatewayTarget(cluster *arov1alpha1.Cluster) (telemetryGatewayTargetSpec, bool, error) {
	if cluster.Spec.GatewayPrivateEndpointIP == "" {
		return telemetryGatewayTargetSpec{}, false, nil
	}
	gatewayPrivateEndpointIP := net.ParseIP(cluster.Spec.GatewayPrivateEndpointIP)
	if gatewayPrivateEndpointIP == nil {
		return telemetryGatewayTargetSpec{}, false, fmt.Errorf("invalid cluster spec field %q: %q is not a valid IP address", "gatewayPrivateEndpointIP", cluster.Spec.GatewayPrivateEndpointIP)
	}

	if cluster.Spec.GatewayTelemetryDomain == "" {
		return telemetryGatewayTargetSpec{
			endpoint: net.JoinHostPort(gatewayPrivateEndpointIP.String(), "4317"),
		}, true, nil
	}

	gatewayHostname := cluster.Spec.GatewayTelemetryDomain
	return telemetryGatewayTargetSpec{
		endpoint: net.JoinHostPort(gatewayHostname, "4317"),
		hostAliases: []corev1.HostAlias{
			{
				IP:        gatewayPrivateEndpointIP.String(),
				Hostnames: []string{gatewayHostname},
			},
		},
	}, true, nil
}

func (r *Reconciler) otelDaemonSets(cluster *arov1alpha1.Cluster, gatewayEndpoint string, hostAliases []corev1.HostAlias) ([]*appsv1.DaemonSet, error) {
	otelPullspec := cluster.Spec.OperatorFlags.GetWithDefault(controllerOTelPullSpec, "")
	if otelPullspec == "" {
		otelPullspec = version.OTelImage(cluster.Spec.ACRDomain)
	}

	newDaemonSet := func(name string, cpuLimit string, nodeSelectorTerms []corev1.NodeSelectorTerm, configKey string) *appsv1.DaemonSet {
		return &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: kubeNamespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": name},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": name},
					},
					Spec: corev1.PodSpec{
						PriorityClassName:            "system-cluster-critical",
						ServiceAccountName:           "geneva",
						AutomountServiceAccountToken: pointerutils.ToPtr(false),
						HostAliases:                  hostAliases,
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: nodeSelectorTerms,
								},
							},
						},
						Tolerations: []corev1.Toleration{
							{
								Effect:   corev1.TaintEffectNoExecute,
								Operator: corev1.TolerationOpExists,
							},
							{
								Effect:   corev1.TaintEffectNoSchedule,
								Operator: corev1.TolerationOpExists,
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "log",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/var/log",
									},
								},
							},
							{
								Name: "otel-file-storage",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/var/lib/otelcol/file_storage",
										Type: pointerutils.ToPtr(corev1.HostPathDirectoryOrCreate),
									},
								},
							},
							{
								Name: "otel-config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: otelConfigMapName,
										},
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "otel-collector",
								Image: otelPullspec,
								Args:  []string{"--config", "/etc/otel/" + configKey},
								Env: []corev1.EnvVar{
									{
										Name:  "GENEVA_GATEWAY_ENDPOINT",
										Value: gatewayEndpoint,
									},
									{
										Name: "MONITORING_ROLE_INSTANCE",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "spec.nodeName",
											},
										},
									},
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "health",
										ContainerPort: 13133,
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.FromInt(13133),
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       10,
									FailureThreshold:    3,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.FromInt(13133),
										},
									},
									InitialDelaySeconds: 5,
									PeriodSeconds:       10,
									FailureThreshold:    3,
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse(cpuLimit),
										corev1.ResourceMemory: resource.MustParse("1000Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("10m"),
										corev1.ResourceMemory: resource.MustParse("250Mi"),
									},
								},
								SecurityContext: &corev1.SecurityContext{
									RunAsUser:                pointerutils.ToPtr(int64(0)),
									AllowPrivilegeEscalation: pointerutils.ToPtr(false),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
									SELinuxOptions: &corev1.SELinuxOptions{
										Type: "spc_t",
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "log",
										ReadOnly:  true,
										MountPath: "/var/log",
									},
									{
										Name:      "otel-config",
										ReadOnly:  true,
										MountPath: "/etc/otel",
									},
									{
										Name:      "otel-file-storage",
										ReadOnly:  false,
										MountPath: "/var/lib/otelcol/file_storage",
									},
								},
							},
						},
					},
				},
			},
		}
	}

	isMasterTerm := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      masterRoleLabel,
				Operator: corev1.NodeSelectorOpExists,
			},
		},
	}
	isControlPlaneTerm := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      controlPlaneRoleLabel,
				Operator: corev1.NodeSelectorOpExists,
			},
		},
	}
	notMasterOrControlPlaneTerm := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      masterRoleLabel,
				Operator: corev1.NodeSelectorOpDoesNotExist,
			},
			{
				Key:      controlPlaneRoleLabel,
				Operator: corev1.NodeSelectorOpDoesNotExist,
			},
		},
	}

	return []*appsv1.DaemonSet{
		newDaemonSet("otel-collector-master", "300m", []corev1.NodeSelectorTerm{isMasterTerm, isControlPlaneTerm}, otelMasterConfigKey),
		newDaemonSet("otel-collector-worker", "200m", []corev1.NodeSelectorTerm{notMasterOrControlPlaneTerm}, otelWorkerConfigKey),
	}, nil
}
