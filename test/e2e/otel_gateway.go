package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	utilcluster "github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	otelE2EVnetName            = "dev-vnet"
	otelE2EContainerName       = "collector"
	otelE2ETelemetryDomain     = "telemetry.e2e.aro.local"
	otelE2ESinkMarker          = "[ARO-E2E-OTEL]"
	otelE2ECollectorConfigEnv  = "OTEL_CONFIG"
	otelE2EImageEnv            = "TELEMETRY_COLLECTOR_IMAGE"
	otelE2ERegistryUserEnv     = "TELEMETRY_COLLECTOR_REGISTRY_USER"
	otelE2ERegistryPasswordEnv = "TELEMETRY_COLLECTOR_REGISTRY_PASSWORD"
	otelE2ELoggingNS           = "openshift-azure-logging"
)

// OTEL gateway e2e (Tier A): point cluster otel-exporter DaemonSets at a
// synthetic telemetrycollector running as ACI in the cluster VNet, inject
// markers for containers/journald/audit, and assert them in ACI debug logs.
//
// Opt-in: skipped unless TELEMETRY_COLLECTOR_IMAGE is set (image must be
// pullable by ACI; registry auth defaults to AZURE_CLIENT_ID/SECRET).
var _ = Describe("OTEL gateway integration", func() {
	var (
		subnetName         string
		containerGroupName string
		originalGatewayIP  string
		originalGatewayDom string
		originalInsecure   string
		hadInsecureFlag    bool
		aciCreated         bool
		subnetCreated      bool
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
			waitForOTelExporterDaemonSetsReady(ctx, "")
		}

		if aciCreated {
			By("deleting ACI container group")
			err := clients.ContainerGroups.DeleteAndWait(ctx, vnetResourceGroup, containerGroupName, nil)
			Expect(err).NotTo(HaveOccurred())
		}

		if subnetCreated {
			By("deleting ACI delegated subnet")
			err := clients.Subnet.DeleteAndWait(ctx, vnetResourceGroup, otelE2EVnetName, subnetName, nil)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("forwards containers, journald, and audit logs to a synthetic ACI collector", func(ctx context.Context) {
		image := os.Getenv(otelE2EImageEnv)
		if image == "" {
			Skip(fmt.Sprintf("%s not set; skipping OTEL ACI e2e", otelE2EImageEnv))
		}

		subnetName = clusterName + "-otel-aci"
		containerGroupName = truncateAzureName("otel-e2e-"+clusterName, 63)

		By("creating ACI-delegated subnet on cluster VNet")
		subnetID, err := createOTelACIISubnet(ctx, subnetName)
		Expect(err).NotTo(HaveOccurred())
		subnetCreated = true

		By("deploying synthetic telemetrycollector ACI")
		collectorConfig, err := staticResources.ReadFile("static_resources/otel-collector.yaml")
		Expect(err).NotTo(HaveOccurred())
		privateIP, err := deployOTelCollectorACI(ctx, containerGroupName, subnetID, image, string(collectorConfig))
		Expect(err).NotTo(HaveOccurred())
		aciCreated = true
		Expect(net.ParseIP(privateIP)).NotTo(BeNil())
		log.Infof("ACI collector private IP: %s", privateIP)

		By("waiting for ACI collector process to be healthy before pointing exporters at it")
		waitForOTelCollectorACIHealthy(ctx, containerGroupName)

		By("saving and patching Cluster CR to point exporters at ACI")
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
			co.Spec.GatewayPrivateEndpointIP = privateIP
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
		waitForOTelExporterDaemonSetsReady(ctx, privateIP)

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
			By(fmt.Sprintf("%s: inject marker %q and assert in ACI logs", src.name, marker))
			src.inject(ctx, marker)
			assertOTelSourceInACILogs(ctx, containerGroupName, marker, src.name, src.extra...)
		}
	})
})

func truncateAzureName(name string, max int) string {
	if len(name) <= max {
		return name
	}
	return name[:max]
}

func createOTelACIISubnet(ctx context.Context, subnetName string) (string, error) {
	prefix, err := pickFreeSubnetPrefix(ctx, vnetResourceGroup, otelE2EVnetName, 28)
	if err != nil {
		return "", err
	}

	subnet := armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: new(prefix),
			Delegations: []*armnetwork.Delegation{
				{
					Name: new("aci-delegation"),
					Properties: &armnetwork.ServiceDelegationPropertiesFormat{
						ServiceName: new("Microsoft.ContainerInstance/containerGroups"),
					},
				},
			},
		},
	}

	if err := clients.Subnet.CreateOrUpdateAndWait(ctx, vnetResourceGroup, otelE2EVnetName, subnetName, subnet, nil); err != nil {
		return "", fmt.Errorf("create ACI subnet %s: %w", subnetName, err)
	}

	resp, err := clients.Subnet.Get(ctx, vnetResourceGroup, otelE2EVnetName, subnetName, nil)
	if err != nil {
		return "", fmt.Errorf("get ACI subnet %s: %w", subnetName, err)
	}
	if resp.ID == nil || *resp.ID == "" {
		return "", fmt.Errorf("ACI subnet %s has empty resource ID", subnetName)
	}
	return *resp.ID, nil
}

func pickFreeSubnetPrefix(ctx context.Context, resourceGroup, vnetName string, prefixLen int) (string, error) {
	existing, err := clients.Subnet.List(ctx, resourceGroup, vnetName, nil)
	if err != nil {
		return "", fmt.Errorf("list subnets: %w", err)
	}
	var used []*net.IPNet
	for _, s := range existing {
		used = append(used, utilcluster.GetIPRangesFromSubnet(s)...)
	}
	return utilcluster.PickFreeDevVNetPrefix(used, prefixLen, utilcluster.GenerateSubnetMaxTries, os.Getenv("CI") != "")
}

func deployOTelCollectorACI(ctx context.Context, groupName, subnetID, image, collectorConfig string) (string, error) {
	registryServer, err := registryServerFromImage(image)
	if err != nil {
		return "", err
	}
	registryUser := firstNonEmpty(os.Getenv(otelE2ERegistryUserEnv), os.Getenv("AZURE_CLIENT_ID"))
	registryPassword := firstNonEmpty(os.Getenv(otelE2ERegistryPasswordEnv), os.Getenv("AZURE_CLIENT_SECRET"))
	if registryUser == "" || registryPassword == "" {
		return "", fmt.Errorf("registry credentials required to pull %s (set %s/%s or AZURE_CLIENT_ID/AZURE_CLIENT_SECRET)", image, otelE2ERegistryUserEnv, otelE2ERegistryPasswordEnv)
	}

	port4317 := int32(4317)
	cg := armcontainerinstance.ContainerGroup{
		Location: new(_env.Location()),
		Properties: &armcontainerinstance.ContainerGroupProperties{
			OSType:        new(armcontainerinstance.OperatingSystemTypesLinux),
			RestartPolicy: new(armcontainerinstance.ContainerGroupRestartPolicyAlways),
			IPAddress: &armcontainerinstance.IPAddress{
				Type: new(armcontainerinstance.ContainerGroupIPAddressTypePrivate),
				Ports: []*armcontainerinstance.Port{
					{
						Port:     new(port4317),
						Protocol: new(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
					},
				},
			},
			SubnetIDs: []*armcontainerinstance.ContainerGroupSubnetID{
				{ID: new(subnetID)},
			},
			ImageRegistryCredentials: []*armcontainerinstance.ImageRegistryCredential{
				{
					Server:   new(registryServer),
					Username: new(registryUser),
					Password: new(registryPassword),
				},
			},
			Containers: []*armcontainerinstance.Container{
				{
					Name: new(otelE2EContainerName),
					Properties: &armcontainerinstance.ContainerProperties{
						Image: new(image),
						Command: []*string{
							new("telemetrycollector"),
							new("--config"),
							new("env:" + otelE2ECollectorConfigEnv),
						},
						EnvironmentVariables: []*armcontainerinstance.EnvironmentVariable{
							{
								Name:        new(otelE2ECollectorConfigEnv),
								SecureValue: new(collectorConfig),
							},
						},
						Ports: []*armcontainerinstance.ContainerPort{
							{
								Port:     new(port4317),
								Protocol: new(armcontainerinstance.ContainerNetworkProtocolTCP),
							},
						},
						Resources: &armcontainerinstance.ResourceRequirements{
							Requests: &armcontainerinstance.ResourceRequests{
								CPU:        new(1.0),
								MemoryInGB: new(1.5),
							},
							Limits: &armcontainerinstance.ResourceLimits{
								CPU:        new(1.0),
								MemoryInGB: new(1.5),
							},
						},
					},
				},
			},
		},
		Tags: map[string]*string{
			"aro-e2e": new("otel-gateway"),
		},
	}

	resp, err := clients.ContainerGroups.CreateOrUpdateAndWait(ctx, vnetResourceGroup, groupName, cg, nil)
	if err != nil {
		return "", fmt.Errorf("create ACI %s: %w", groupName, err)
	}
	if resp.Properties == nil || resp.Properties.IPAddress == nil || resp.Properties.IPAddress.IP == nil {
		return "", fmt.Errorf("ACI %s created without private IP", groupName)
	}
	return *resp.Properties.IPAddress.IP, nil
}

func waitForOTelCollectorACIHealthy(ctx context.Context, containerGroupName string) {
	GinkgoHelper()

	Eventually(func(g Gomega, ctx context.Context) {
		logs, err := clients.Containers.ListLogs(ctx, vnetResourceGroup, containerGroupName, otelE2EContainerName, &armcontainerinstance.ContainersClientListLogsOptions{
			Tail: new(int32(100)),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logs.Content).NotTo(BeNil())
		g.Expect(*logs.Content).To(ContainSubstring("Everything is ready"),
			"ACI collector not ready; logs:\n%s", *logs.Content)
	}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}

func registryServerFromImage(image string) (string, error) {
	// Images are host/path:tag — not full URLs.
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

// waitForOTelExporterDaemonSetsReady waits until master/worker otel-exporter
// DaemonSets are rolled out. When wantHostAliasIP is set, also require that IP
// in each DS hostAliases (gateway target after the Cluster CR patch).
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
			g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0))
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberAvailable))
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.UpdatedNumberScheduled))
			g.Expect(ds.Status.CurrentNumberScheduled).To(Equal(ds.Status.NumberReady))
			g.Expect(ds.Generation).To(Equal(ds.Status.ObservedGeneration))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func otelHostAliasContains(aliases []corev1.HostAlias, ip string) bool {
	for _, ha := range aliases {
		if ha.IP == ip {
			return true
		}
	}
	return false
}

func assertOTelSourceInACILogs(ctx context.Context, containerGroupName, marker, source string, extra ...string) {
	GinkgoHelper()
	want := append([]string{marker, otelE2ESinkMarker, "source_name: Str(" + source + ")", "EventName: Str(" + source + ")"}, extra...)
	Eventually(func(g Gomega, ctx context.Context) {
		logs, err := clients.Containers.ListLogs(ctx, vnetResourceGroup, containerGroupName, otelE2EContainerName, &armcontainerinstance.ContainersClientListLogsOptions{
			Tail: new(int32(400)),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logs.Content).NotTo(BeNil())
		for _, s := range want {
			g.Expect(*logs.Content).To(ContainSubstring(s), "missing %q for source %s:\n%s", s, source, *logs.Content)
		}
	}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}

func injectOTelContainerMarker(ctx context.Context, marker string) {
	GinkgoHelper()
	// Platform NS + level=error: required by minimal-logs file_log keep filters.
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
	// user.err => PRIORITY=3, kept by minimal-logs journald high-signal filter.
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
