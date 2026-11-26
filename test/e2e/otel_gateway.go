package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
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

	"sigs.k8s.io/yaml"

	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	otelE2ECollectorNS        = "aro-e2e-otel"
	otelE2ECollectorName      = "otel-e2e-collector"
	otelE2ECollectorContainer = "collector"
	otelE2ECollectorMarker    = "[ARO-E2E-OTEL]"
	otelE2EImageEnv           = "TELEMETRY_COLLECTOR_IMAGE"
	otelE2ELoggingNS          = "openshift-azure-logging"
	otelE2EOTLPPort           = int32(4317)

	otelE2EExporterMaster     = "aro-e2e-otel-exporter-master"
	otelE2EExporterWorker     = "aro-e2e-otel-exporter-worker"
	otelE2EExporterConfigMap  = "aro-e2e-otel-exporter-config"
	otelE2EOTelConfigMap      = "otel-config"
	otelE2EMasterConfigKey    = "master-config.yaml"
	otelE2EWorkerConfigKey    = "worker-config.yaml"
	otelE2EGatewayExporterID  = "otlp_grpc/gateway"
	otelE2EGatewayEndpointEnv = "GENEVA_GATEWAY_ENDPOINT"
	otelE2EProdFileStorage    = "/var/lib/otelcol/file_storage"
	otelE2EProdContainer      = "otel-exporter"
	otelE2EExporterContainer  = "otel-exporter-e2e"
)

var otelE2EExporterClones = []struct {
	src       string
	name      string
	configKey string
}{
	{genevalogging.MasterDaemonsetName, otelE2EExporterMaster, otelE2EMasterConfigKey},
	{genevalogging.WorkerDaemonsetName, otelE2EExporterWorker, otelE2EWorkerConfigKey},
}

var _ = Describe("OTEL gateway integration", func() {
	var (
		collectorCreated bool
		exporterCreated  bool
	)

	AfterEach(func(ctx context.Context) {
		if exporterCreated {
			By("deleting test-owned otel exporters")
			deleteOTelE2EExporters(ctx)
		}

		if collectorCreated {
			By("deleting in-cluster OTEL collector namespace")
			deleteOTelE2ECollector(ctx)
		}
	})

	It("forwards containers, journald, and audit logs to an in-cluster collector", func(ctx context.Context) {
		image := os.Getenv(otelE2EImageEnv)
		if image == "" {
			Skip(fmt.Sprintf("%s not set; skipping OTEL e2e", otelE2EImageEnv))
		}

		By("deleting leftovers from previous runs")
		deleteOTelE2EExporters(ctx)
		deleteOTelE2ECollector(ctx)

		By("deploying in-cluster synthetic telemetrycollector")
		collectorConfig, err := staticResources.ReadFile("static_resources/otel-collector.yaml")
		Expect(err).NotTo(HaveOccurred())
		collectorCreated = true
		collectorClusterIP, err := deployOTelCollectorInCluster(ctx, image, string(collectorConfig))
		Expect(err).NotTo(HaveOccurred())
		Expect(net.ParseIP(collectorClusterIP)).NotTo(BeNil())
		log.Infof("in-cluster collector Service ClusterIP: %s", collectorClusterIP)

		By("waiting for collector to be healthy before exporting to it")
		waitForOTelCollectorPodHealthy(ctx)

		By("waiting for the operator to have rendered its otel exporters")
		waitForOperatorOTelExporters(ctx)

		By("deploying test-owned copies of the operator-rendered otel exporters")
		exporterCreated = true
		endpoint := net.JoinHostPort(collectorClusterIP, strconv.Itoa(int(otelE2EOTLPPort)))
		Expect(deployOTelE2EExporter(ctx, endpoint)).To(Succeed())
		waitForOTelE2EExporterReady(ctx)

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

func deleteOTelE2EExporters(ctx context.Context) {
	GinkgoHelper()
	for _, clone := range otelE2EExporterClones {
		CleanupK8sResource[*appsv1.DaemonSet](ctx, clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS), clone.name)
	}
	CleanupK8sResource[*corev1.ConfigMap](ctx, clients.Kubernetes.CoreV1().ConfigMaps(otelE2ELoggingNS), otelE2EExporterConfigMap)
}

func deleteOTelE2ECollector(ctx context.Context) {
	GinkgoHelper()
	CleanupK8sResource[*corev1.Namespace](ctx, clients.Kubernetes.CoreV1().Namespaces(), otelE2ECollectorNS)
}

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

	labels := map[string]string{"app": otelE2ECollectorName}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: otelE2ECollectorName, Namespace: otelE2ECollectorNS},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
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

func deployOTelE2EExporter(ctx context.Context, endpoint string) error {
	cm, err := clients.Kubernetes.CoreV1().ConfigMaps(otelE2ELoggingNS).Get(ctx, otelE2EOTelConfigMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get %s configmap: %w", otelE2EOTelConfigMap, err)
	}

	patched := map[string]string{}
	for _, clone := range otelE2EExporterClones {
		rendered, ok := cm.Data[clone.configKey]
		if !ok {
			return fmt.Errorf("configmap %s has no %s key", otelE2EOTelConfigMap, clone.configKey)
		}
		cfg, err := otelE2EExporterConfig(rendered)
		if err != nil {
			return err
		}
		patched[clone.configKey] = cfg
	}

	if _, err := clients.Kubernetes.CoreV1().ConfigMaps(otelE2ELoggingNS).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: otelE2EExporterConfigMap, Namespace: otelE2ELoggingNS},
		Data:       patched,
	}, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create exporter configmap: %w", err)
	}

	for _, clone := range otelE2EExporterClones {
		src, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Get(ctx, clone.src, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get %s daemonset: %w", clone.src, err)
		}
		ds, err := otelE2EExporterDaemonSet(src, clone.name, endpoint)
		if err != nil {
			return err
		}
		if _, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Create(ctx, ds, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create %s daemonset: %w", clone.name, err)
		}
	}
	return nil
}

func waitForOperatorOTelExporters(ctx context.Context) {
	GinkgoHelper()
	for _, clone := range otelE2EExporterClones {
		Eventually(func(g Gomega, ctx context.Context) {
			_, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Get(ctx, clone.src, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred(),
				"operator has not created DaemonSet %s; the gateway target may not be ready", clone.src)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func otelE2EExporterConfig(rendered string) (string, error) {
	var root map[string]any
	if err := yaml.Unmarshal([]byte(rendered), &root); err != nil {
		return "", fmt.Errorf("parse rendered otel config: %w", err)
	}
	exporters, ok := root["exporters"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("rendered otel config has no exporters block")
	}
	gateway, ok := exporters[otelE2EGatewayExporterID].(map[string]any)
	if !ok {
		return "", fmt.Errorf("rendered otel config has no %q exporter", otelE2EGatewayExporterID)
	}
	gateway["tls"] = map[string]any{"insecure": true}

	receivers, ok := root["receivers"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("rendered otel config has no receivers block")
	}
	journald, ok := receivers["journald"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("rendered otel config has no journald receiver")
	}
	journald["start_at"] = "end"

	out, err := yaml.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("render patched otel config: %w", err)
	}
	return string(out), nil
}

func otelE2EExporterDaemonSet(src *appsv1.DaemonSet, name, endpoint string) (*appsv1.DaemonSet, error) {
	ds := src.DeepCopy()
	labels := map[string]string{"app": name}

	ds.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: otelE2ELoggingNS,
		Labels:    labels,
	}
	ds.Status = appsv1.DaemonSetStatus{}
	ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	ds.Spec.Template.Labels = labels
	ds.Spec.Template.Annotations = nil
	ds.Spec.Template.Spec.HostAliases = nil

	var sawConfigVolume, sawStorageVolume bool
	for i := range ds.Spec.Template.Spec.Volumes {
		v := &ds.Spec.Template.Spec.Volumes[i]
		if v.ConfigMap != nil && v.ConfigMap.Name == otelE2EOTelConfigMap {
			v.ConfigMap.Name = otelE2EExporterConfigMap
			sawConfigVolume = true
		}
		if v.HostPath != nil && v.HostPath.Path == otelE2EProdFileStorage {
			v.HostPath = nil
			v.EmptyDir = &corev1.EmptyDirVolumeSource{}
			sawStorageVolume = true
		}
	}
	if !sawConfigVolume {
		return nil, fmt.Errorf("daemonset %s has no %s configmap volume", src.Name, otelE2EOTelConfigMap)
	}
	if !sawStorageVolume {
		return nil, fmt.Errorf("daemonset %s has no %s hostPath volume", src.Name, otelE2EProdFileStorage)
	}

	var sawContainer, sawEndpoint bool
	for i := range ds.Spec.Template.Spec.Containers {
		c := &ds.Spec.Template.Spec.Containers[i]
		if c.Name == otelE2EProdContainer {
			c.Name = otelE2EExporterContainer
			sawContainer = true
		}
		for j := range c.Env {
			if c.Env[j].Name == otelE2EGatewayEndpointEnv {
				c.Env[j].Value = endpoint
				sawEndpoint = true
			}
		}
	}
	if !sawContainer {
		return nil, fmt.Errorf("daemonset %s has no %s container", src.Name, otelE2EProdContainer)
	}
	if !sawEndpoint {
		return nil, fmt.Errorf("daemonset %s has no %s env var", src.Name, otelE2EGatewayEndpointEnv)
	}
	return ds, nil
}

func waitForOTelE2EExporterReady(ctx context.Context) {
	GinkgoHelper()
	for _, clone := range otelE2EExporterClones {
		Eventually(func(g Gomega, ctx context.Context) {
			ds, err := clients.Kubernetes.AppsV1().DaemonSets(otelE2ELoggingNS).Get(ctx, clone.name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0),
				"DaemonSet %s scheduled onto no nodes", clone.name)
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberAvailable))
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.UpdatedNumberScheduled))
			g.Expect(ds.Status.CurrentNumberScheduled).To(Equal(ds.Status.NumberReady))
			g.Expect(ds.Generation).To(Equal(ds.Status.ObservedGeneration))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func assertOTelSourceInCollectorLogs(ctx context.Context, marker, source string, extra ...string) {
	GinkgoHelper()
	want := append([]string{marker, otelE2ECollectorMarker, "source_name: Str(" + source + ")", "EventName: Str(" + source + ")"}, extra...)
	Eventually(func(g Gomega, ctx context.Context) {
		logs, err := otelCollectorPodLogs(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(otelLogRecordContainingAll(logs, want)).To(BeTrue(),
			"no single log record contains all of %q for source %s:\n%s", want, source, logs)
	}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}

var otelDebugExporterBlockHeader = regexp.MustCompile(`(?m)^(?:ResourceLog|ScopeLogs|LogRecord) #\d+`)

func otelLogRecordContainingAll(logs string, want []string) bool {
	starts := otelDebugExporterBlockHeader.FindAllStringIndex(logs, -1)
	for i, start := range starts {
		end := len(logs)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		block := logs[start[0]:end]
		if !strings.HasPrefix(block, "LogRecord #") {
			continue
		}
		matched := true
		for _, s := range want {
			if !strings.Contains(block, s) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
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
