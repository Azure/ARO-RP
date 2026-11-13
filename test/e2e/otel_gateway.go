package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	otelE2ECollectorNS         = "aro-e2e-otel"
	otelE2ECollectorName       = "otel-e2e-collector"
	otelE2ECollectorContainer  = "collector"
	otelE2ETelemetryDomain     = "telemetry.e2e.aro.local"
	otelE2ECollectorMarker     = "[ARO-E2E-OTEL]"
	otelE2EImageEnv            = "TELEMETRY_COLLECTOR_IMAGE"
	otelE2ERegistryUserEnv     = "TELEMETRY_COLLECTOR_REGISTRY_USER"
	otelE2ERegistryPasswordEnv = "TELEMETRY_COLLECTOR_REGISTRY_PASSWORD"
	otelE2ELoggingNS           = "openshift-azure-logging"
	otelE2EOTLPPort            = int32(4317)
)

var _ = Describe("OTEL gateway integration", func() {
	var (
		collectorClusterIP string
		originalGatewayIP  string
		originalGatewayDom string
		originalInsecure   string
		hadInsecureFlag    bool
		collectorCreated   bool
		clusterPatched     bool
	)

	AfterEach(func(ctx context.Context) {
		if clusterPatched {
			By("restoring Cluster CR gateway fields and insecure flag")
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				co.Spec.GatewayPrivateEndpointIP = originalGatewayIP
				co.Spec.GatewayTelemetryDomain = originalGatewayDom
				if co.Spec.OperatorFlags == nil {
					co.Spec.OperatorFlags = arov1alpha1.OperatorFlags{}
				}
				if hadInsecureFlag {
					co.Spec.OperatorFlags[operator.GenevaLoggingOTelGatewayInsecure] = originalInsecure
				} else {
					delete(co.Spec.OperatorFlags, operator.GenevaLoggingOTelGatewayInsecure)
				}
				_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			waitForOTelExporterDaemonSetsRestored(ctx, collectorClusterIP, originalGatewayIP)
		}

		if collectorCreated {
			By("deleting in-cluster OTEL collector namespace")
			err := clients.Kubernetes.CoreV1().Namespaces().Delete(ctx, otelE2ECollectorNS, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("forwards containers, journald, and audit logs to an in-cluster collector", func(ctx context.Context) {
		image := os.Getenv(otelE2EImageEnv)
		if image == "" {
			Skip(fmt.Sprintf("%s not set; skipping OTEL e2e", otelE2EImageEnv))
		}

		By("deploying in-cluster synthetic telemetrycollector")
		collectorConfig, err := staticResources.ReadFile("static_resources/otel-collector.yaml")
		Expect(err).NotTo(HaveOccurred())
		collectorClusterIP, err = deployOTelCollectorInCluster(ctx, image, string(collectorConfig))
		Expect(err).NotTo(HaveOccurred())
		collectorCreated = true
		Expect(net.ParseIP(collectorClusterIP)).NotTo(BeNil())
		log.Infof("in-cluster collector Service ClusterIP: %s", collectorClusterIP)

		By("waiting for collector to be healthy before pointing exporters at it")
		waitForOTelCollectorPodHealthy(ctx)

		By("saving and patching Cluster CR to point exporters at in-cluster collector")
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		originalGatewayIP = co.Spec.GatewayPrivateEndpointIP
		originalGatewayDom = co.Spec.GatewayTelemetryDomain
		originalInsecure, hadInsecureFlag = co.Spec.OperatorFlags[operator.GenevaLoggingOTelGatewayInsecure]

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.GatewayPrivateEndpointIP = collectorClusterIP
			co.Spec.GatewayTelemetryDomain = otelE2ETelemetryDomain
			if co.Spec.OperatorFlags == nil {
				co.Spec.OperatorFlags = arov1alpha1.OperatorFlags{}
			}
			co.Spec.OperatorFlags[operator.GenevaLoggingOTelGatewayInsecure] = operator.FlagTrue
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())
		clusterPatched = true

		By("waiting for otel-exporter DaemonSets to roll out with new gateway target")
		waitForOTelConfigInsecure(ctx)
		waitForOTelExporterDaemonSetsReady(ctx, collectorClusterIP)

		marker := "aro-e2e-otel-marker-" + strings.ToLower(uuid.DefaultGenerator.Generate())
		for _, src := range []struct {
			name   string
			inject func(context.Context, string)
			extra  []string
		}{
			{"containers", injectOTelContainerMarker, []string{
				`namespace: Str(` + otelE2ELoggingNS + `)`,
				`container: Str(logger)`,
			}},
			{"journald", injectOTelJournaldMarker, []string{`SYSLOG_IDENTIFIER: Str(aro-e2e-otel)`}},
			{"audit", injectOTelAuditMarker, nil},
		} {
			By(fmt.Sprintf("%s: inject marker %q and assert in collector logs", src.name, marker))
			src.inject(ctx, marker)
			assertOTelSourceInCollectorLogs(ctx, marker, src.name, src.extra...)
		}
	})
})

func deployOTelCollectorInCluster(ctx context.Context, image, collectorConfig string) (string, error) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otelE2ECollectorNS}}
	if _, err := clients.Kubernetes.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("create namespace %s: %w", otelE2ECollectorNS, err)
	}

	if _, err := clients.Kubernetes.CoreV1().ConfigMaps(otelE2ECollectorNS).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: otelE2ECollectorName, Namespace: otelE2ECollectorNS},
		Data:       map[string]string{"config.yaml": collectorConfig},
	}, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("create collector configmap: %w", err)
	}

	var pullSecrets []corev1.LocalObjectReference
	if secretName, err := maybeCreateOTelPullSecret(ctx, image); err != nil {
		return "", err
	} else if secretName != "" {
		pullSecrets = []corev1.LocalObjectReference{{Name: secretName}}
	}

	labels := map[string]string{"app": otelE2ECollectorName}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: otelE2ECollectorName, Namespace: otelE2ECollectorNS},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ImagePullSecrets: pullSecrets,
					Containers: []corev1.Container{{
						Name:  otelE2ECollectorContainer,
						Image: image,
						Args:  []string{"--config", "/etc/otel/config.yaml"},
						Ports: []corev1.ContainerPort{{
							Name: "otlp-grpc", ContainerPort: otelE2EOTLPPort, Protocol: corev1.ProtocolTCP,
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name: "config", MountPath: "/etc/otel", ReadOnly: true,
						}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt32(13133),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       5,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: otelE2ECollectorName},
							},
						},
					}},
				},
			},
		},
	}
	if _, err := clients.Kubernetes.AppsV1().Deployments(otelE2ECollectorNS).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("create collector deployment: %w", err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: otelE2ECollectorName, Namespace: otelE2ECollectorNS},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name: "otlp-grpc", Port: otelE2EOTLPPort, TargetPort: intstr.FromInt32(otelE2EOTLPPort), Protocol: corev1.ProtocolTCP,
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	created, err := clients.Kubernetes.CoreV1().Services(otelE2ECollectorNS).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create collector service: %w", err)
	}
	if created.Spec.ClusterIP == "" {
		return "", fmt.Errorf("collector service has empty ClusterIP")
	}
	return created.Spec.ClusterIP, nil
}

func maybeCreateOTelPullSecret(ctx context.Context, image string) (string, error) {
	user := firstNonEmpty(os.Getenv(otelE2ERegistryUserEnv), os.Getenv("AZURE_CLIENT_ID"))
	pass := firstNonEmpty(os.Getenv(otelE2ERegistryPasswordEnv), os.Getenv("AZURE_CLIENT_SECRET"))
	if user == "" || pass == "" {
		return "", nil
	}
	server, err := registryServerFromImage(image)
	if err != nil {
		return "", err
	}
	dockerConfig, err := json.Marshal(map[string]any{
		"auths": map[string]any{
			server: map[string]string{
				"username": user,
				"password": pass,
			},
		},
	})
	if err != nil {
		return "", err
	}
	secretName := otelE2ECollectorName + "-pull"
	_, err = clients.Kubernetes.CoreV1().Secrets(otelE2ECollectorNS).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: otelE2ECollectorNS},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{corev1.DockerConfigJsonKey: dockerConfig},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create pull secret: %w", err)
	}
	return secretName, nil
}

func waitForOTelCollectorPodHealthy(ctx context.Context) {
	GinkgoHelper()
	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := clients.Kubernetes.CoreV1().Pods(otelE2ECollectorNS).List(ctx, metav1.ListOptions{
			LabelSelector: "app=" + otelE2ECollectorName,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pods.Items).NotTo(BeEmpty())
		pod := pods.Items[0]
		g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
		ready := false
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		g.Expect(ready).To(BeTrue(), "collector pod not Ready: %+v", pod.Status)
	}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
}

func registryServerFromImage(image string) (string, error) {
	hostPath := image
	if strings.Contains(image, "://") {
		u, err := url.Parse(image)
		if err != nil {
			return "", err
		}
		hostPath = strings.TrimPrefix(u.Host+u.Path, "/")
	}
	parts := strings.Split(hostPath, "/")
	if len(parts) == 0 || parts[0] == "" || !strings.Contains(parts[0], ".") {
		return "", fmt.Errorf("cannot derive registry server from image %q", image)
	}
	return parts[0], nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func waitForOTelConfigInsecure(ctx context.Context) {
	GinkgoHelper()
	Eventually(func(g Gomega, ctx context.Context) {
		cm, err := clients.Kubernetes.CoreV1().ConfigMaps(otelE2ELoggingNS).Get(ctx, "otel-config", metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cm.Data["master-config.yaml"] + cm.Data["worker-config.yaml"]).To(ContainSubstring("insecure: true"))
	}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
}

func waitForOTelExporterDaemonSetsReady(ctx context.Context, wantHostAliasIP string) {
	GinkgoHelper()
	for _, daemonSetName := range []string{genevalogging.MasterDaemonsetName, genevalogging.WorkerDaemonsetName} {
		Eventually(func(g Gomega, ctx context.Context) {
			ds, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Get(ctx, daemonSetName, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			if wantHostAliasIP != "" {
				g.Expect(otelHostAliasContains(ds.Spec.Template.Spec.HostAliases, wantHostAliasIP)).To(BeTrue(),
					"DaemonSet %s missing hostAlias for %s", daemonSetName, wantHostAliasIP)
			}
			assertOTelExporterDaemonSetRolledOut(g, ds)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func waitForOTelExporterDaemonSetsRestored(ctx context.Context, leaveIP, restoreIP string) {
	GinkgoHelper()
	for _, daemonSetName := range []string{genevalogging.MasterDaemonsetName, genevalogging.WorkerDaemonsetName} {
		Eventually(func(g Gomega, ctx context.Context) {
			ds, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Get(ctx, daemonSetName, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			if leaveIP != "" {
				g.Expect(otelHostAliasContains(ds.Spec.Template.Spec.HostAliases, leaveIP)).To(BeFalse(),
					"DaemonSet %s still has hostAlias for collector %s", daemonSetName, leaveIP)
			}
			if restoreIP != "" {
				g.Expect(otelHostAliasContains(ds.Spec.Template.Spec.HostAliases, restoreIP)).To(BeTrue(),
					"DaemonSet %s missing restored hostAlias for %s", daemonSetName, restoreIP)
			}
			assertOTelExporterDaemonSetRolledOut(g, ds)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func assertOTelExporterDaemonSetRolledOut(g Gomega, ds *appsv1.DaemonSet) {
	GinkgoHelper()
	g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0))
	g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberAvailable))
	g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.UpdatedNumberScheduled))
	g.Expect(ds.Status.CurrentNumberScheduled).To(Equal(ds.Status.NumberReady))
	g.Expect(ds.Generation).To(Equal(ds.Status.ObservedGeneration))
}

func otelHostAliasContains(aliases []corev1.HostAlias, ip string) bool {
	for _, ha := range aliases {
		if ha.IP == ip {
			return true
		}
	}
	return false
}

func assertOTelSourceInCollectorLogs(ctx context.Context, marker, source string, extra ...string) {
	GinkgoHelper()
	want := append([]string{marker, otelE2ECollectorMarker, "source_name: Str(" + source + ")", "EventName: Str(" + source + ")"}, extra...)
	Eventually(func(g Gomega, ctx context.Context) {
		logs, err := otelCollectorPodLogs(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		for _, s := range want {
			g.Expect(logs).To(ContainSubstring(s), "missing %q for source %s:\n%s", s, source, logs)
		}
	}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}

func otelCollectorPodLogs(ctx context.Context) (string, error) {
	pods, err := clients.Kubernetes.CoreV1().Pods(otelE2ECollectorNS).List(ctx, metav1.ListOptions{
		LabelSelector: "app=" + otelE2ECollectorName,
	})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no collector pods in %s", otelE2ECollectorNS)
	}
	req := clients.Kubernetes.CoreV1().Pods(otelE2ECollectorNS).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{
		Container: otelE2ECollectorContainer,
		TailLines: new(int64(400)),
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	b, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func injectOTelContainerMarker(ctx context.Context, marker string) {
	GinkgoHelper()
	ns := otelE2ELoggingNS
	podName := "otel-marker-" + strings.ToLower(uuid.DefaultGenerator.Generate()[:8])
	_, err := clients.Kubernetes.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{{
				Name: "logger", Image: "busybox",
				Command: []string{"/bin/sh", "-c", fmt.Sprintf("while true; do echo %q; sleep 5; done", "level=error "+marker)},
			}},
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx context.Context) {
		_ = clients.Kubernetes.CoreV1().Pods(ns).Delete(ctx, podName, metav1.DeleteOptions{})
	})
	Eventually(func(g Gomega, ctx context.Context) {
		p, err := clients.Kubernetes.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(p.Status.Phase).To(Equal(corev1.PodRunning))
	}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
}

func injectOTelJournaldMarker(ctx context.Context, marker string) {
	GinkgoHelper()
	pods, err := clients.Kubernetes.CoreV1().Pods("openshift-machine-config-operator").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=machine-config-daemon",
		FieldSelector: "status.phase=Running",
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(pods.Items).NotTo(BeEmpty())
	Expect(execInPod(ctx, pods.Items[0].Namespace, pods.Items[0].Name, "machine-config-daemon",
		[]string{"chroot", "/rootfs", "logger", "-p", "user.err", "-t", "aro-e2e-otel", marker},
	)).To(Succeed())
}

func injectOTelAuditMarker(ctx context.Context, marker string) {
	GinkgoHelper()
	ns := otelE2ELoggingNS
	_, err := clients.Kubernetes.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: marker, Namespace: ns},
		Data:       map[string]string{"marker": marker},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx context.Context) {
		_ = clients.Kubernetes.CoreV1().ConfigMaps(ns).Delete(ctx, marker, metav1.DeleteOptions{})
	})
}

func execInPod(ctx context.Context, namespace, pod, container string, command []string) error {
	req := clients.Kubernetes.CoreV1().RESTClient().Post().
		Resource("pods").Name(pod).Namespace(namespace).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{Container: container, Command: command, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(clients.RestConfig, "POST", req.URL())
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr}); err != nil {
		return fmt.Errorf("%w: stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	return nil
}
