package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	utilcluster "github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
)

// OTEL gateway e2e (Tier A): point cluster otel-exporter DaemonSets at a
// synthetic telemetrycollector running as ACI in the cluster VNet, inject a
// unique high-signal container log, and assert the marker appears in ACI logs.
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
			err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
				co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
				if err != nil {
					return false, err
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
				if kerrors.IsConflict(err) {
					return false, nil
				}
				return err == nil, err
			})
			Expect(err).NotTo(HaveOccurred())
			waitForOTelExporterDaemonSetsReady(ctx)
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

	It("forwards cluster container logs to a synthetic ACI collector", func(ctx context.Context) {
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

		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			co.Spec.GatewayPrivateEndpointIP = privateIP
			co.Spec.GatewayTelemetryDomain = otelE2ETelemetryDomain
			if co.Spec.OperatorFlags == nil {
				co.Spec.OperatorFlags = arov1alpha1.OperatorFlags{}
			}
			co.Spec.OperatorFlags[operator.GenevaLoggingOTelGatewayInsecure] = operator.FlagTrue
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
			if kerrors.IsConflict(err) {
				return false, nil
			}
			return err == nil, err
		})
		Expect(err).NotTo(HaveOccurred())
		clusterPatched = true

		By("waiting for otel-exporter DaemonSets to roll out with new gateway target")
		Eventually(func(g Gomega, ctx context.Context) {
			cm, err := clients.Kubernetes.CoreV1().ConfigMaps(genevaloggingNamespace()).Get(ctx, "otel-config", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cm.Data["master-config.yaml"] + cm.Data["worker-config.yaml"]).To(ContainSubstring("insecure: true"))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
		Eventually(func(g Gomega, ctx context.Context) {
			for _, daemonSetName := range []string{genevalogging.MasterDaemonsetName, genevalogging.WorkerDaemonsetName} {
				ds, err := clients.Kubernetes.AppsV1().DaemonSets(genevaloggingNamespace()).Get(ctx, daemonSetName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				foundIP := false
				for _, ha := range ds.Spec.Template.Spec.HostAliases {
					if ha.IP == privateIP {
						foundIP = true
						break
					}
				}
				g.Expect(foundIP).To(BeTrue(), "DaemonSet %s missing hostAlias for ACI IP %s", daemonSetName, privateIP)
				g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0))
				g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberAvailable))
				g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.UpdatedNumberScheduled))
				g.Expect(ds.Status.CurrentNumberScheduled).To(Equal(ds.Status.NumberReady))
				g.Expect(ds.Generation).To(Equal(ds.Status.ObservedGeneration))
			}
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		marker := fmt.Sprintf("ARO-E2E-OTEL-MARKER-%s", uuid.DefaultGenerator.Generate())
		By(fmt.Sprintf("injecting high-signal container log marker %q", marker))
		injectOTelMarkerPod(ctx, marker)

		By("polling ACI logs for sink attribute and marker")
		Eventually(func(g Gomega, ctx context.Context) {
			logs, err := clients.Containers.ListLogs(ctx, vnetResourceGroup, containerGroupName, otelE2EContainerName, &armcontainerinstance.ContainersClientListLogsOptions{
				Tail: pointerutils.ToPtr(int32(500)),
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(logs.Content).NotTo(BeNil())
			content := *logs.Content
			g.Expect(content).To(ContainSubstring(otelE2ESinkMarker))
			g.Expect(content).To(ContainSubstring(marker))
		}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
	})
})

func genevaloggingNamespace() string {
	return "openshift-azure-logging"
}

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
			AddressPrefix: pointerutils.ToPtr(prefix),
			Delegations: []*armnetwork.Delegation{
				{
					Name: pointerutils.ToPtr("aci-delegation"),
					Properties: &armnetwork.ServiceDelegationPropertiesFormat{
						ServiceName: pointerutils.ToPtr("Microsoft.ContainerInstance/containerGroups"),
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

	// Prefer 10.3.0.0–10.127.x like CI cluster subnet generation; /28 must be aligned.
	for try := 0; try < 256; try++ {
		x := 3 + rand.Intn(125)
		y := rand.Intn(256)
		host := rand.Intn(16) * 16
		candidate := fmt.Sprintf("10.%d.%d.%d/%d", x, y, host, prefixLen)
		_, cidr, err := net.ParseCIDR(candidate)
		if err != nil {
			continue
		}
		if ipNetOverlapsAny(cidr, used) {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("could not find free /%d prefix in %s/%s", prefixLen, resourceGroup, vnetName)
}

func ipNetOverlapsAny(candidate *net.IPNet, used []*net.IPNet) bool {
	for _, u := range used {
		if ipNetsOverlap(candidate, u) {
			return true
		}
	}
	return false
}

func ipNetsOverlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
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
		Location: pointerutils.ToPtr(_env.Location()),
		Properties: &armcontainerinstance.ContainerGroupProperties{
			OSType:        pointerutils.ToPtr(armcontainerinstance.OperatingSystemTypesLinux),
			RestartPolicy: pointerutils.ToPtr(armcontainerinstance.ContainerGroupRestartPolicyAlways),
			IPAddress: &armcontainerinstance.IPAddress{
				Type: pointerutils.ToPtr(armcontainerinstance.ContainerGroupIPAddressTypePrivate),
				Ports: []*armcontainerinstance.Port{
					{
						Port:     pointerutils.ToPtr(port4317),
						Protocol: pointerutils.ToPtr(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
					},
				},
			},
			SubnetIDs: []*armcontainerinstance.ContainerGroupSubnetID{
				{ID: pointerutils.ToPtr(subnetID)},
			},
			ImageRegistryCredentials: []*armcontainerinstance.ImageRegistryCredential{
				{
					Server:   pointerutils.ToPtr(registryServer),
					Username: pointerutils.ToPtr(registryUser),
					Password: pointerutils.ToPtr(registryPassword),
				},
			},
			Containers: []*armcontainerinstance.Container{
				{
					Name: pointerutils.ToPtr(otelE2EContainerName),
					Properties: &armcontainerinstance.ContainerProperties{
						Image: pointerutils.ToPtr(image),
						Command: []*string{
							pointerutils.ToPtr("telemetrycollector"),
							pointerutils.ToPtr("--config"),
							pointerutils.ToPtr("env:" + otelE2ECollectorConfigEnv),
						},
						EnvironmentVariables: []*armcontainerinstance.EnvironmentVariable{
							{
								Name:        pointerutils.ToPtr(otelE2ECollectorConfigEnv),
								SecureValue: pointerutils.ToPtr(collectorConfig),
							},
						},
						Ports: []*armcontainerinstance.ContainerPort{
							{
								Port:     pointerutils.ToPtr(port4317),
								Protocol: pointerutils.ToPtr(armcontainerinstance.ContainerNetworkProtocolTCP),
							},
						},
						Resources: &armcontainerinstance.ResourceRequirements{
							Requests: &armcontainerinstance.ResourceRequests{
								CPU:        pointerutils.ToPtr(1.0),
								MemoryInGB: pointerutils.ToPtr(1.5),
							},
							Limits: &armcontainerinstance.ResourceLimits{
								CPU:        pointerutils.ToPtr(1.0),
								MemoryInGB: pointerutils.ToPtr(1.5),
							},
						},
					},
				},
			},
		},
		Tags: map[string]*string{
			"aro-e2e": pointerutils.ToPtr("otel-gateway"),
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

// waitForOTelCollectorACIHealthy waits until the collector process has started
// successfully before exporters are pointed at it.
func waitForOTelCollectorACIHealthy(ctx context.Context, containerGroupName string) {
	GinkgoHelper()

	Eventually(func(g Gomega, ctx context.Context) {
		logs, err := clients.Containers.ListLogs(ctx, vnetResourceGroup, containerGroupName, otelE2EContainerName, &armcontainerinstance.ContainersClientListLogsOptions{
			Tail: pointerutils.ToPtr(int32(100)),
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

func waitForOTelExporterDaemonSetsReady(ctx context.Context) {
	GinkgoHelper()
	for _, daemonSetName := range []string{genevalogging.MasterDaemonsetName, genevalogging.WorkerDaemonsetName} {
		Eventually(func(g Gomega, ctx context.Context) {
			ds, err := clients.Kubernetes.AppsV1().DaemonSets(genevaloggingNamespace()).Get(ctx, daemonSetName, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ds.Status.DesiredNumberScheduled).To(BeNumerically(">", 0))
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.NumberAvailable))
			g.Expect(ds.Status.DesiredNumberScheduled).To(Equal(ds.Status.UpdatedNumberScheduled))
			g.Expect(ds.Status.CurrentNumberScheduled).To(Equal(ds.Status.NumberReady))
			g.Expect(ds.Generation).To(Equal(ds.Status.ObservedGeneration))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}
}

func injectOTelMarkerPod(ctx context.Context, marker string) {
	GinkgoHelper()
	// minimal-logs file_log/containers only scrapes platform namespaces
	// (openshift-azure-*, control-plane, etc.) — never customer workloads.
	ns := genevaloggingNamespace()

	// minimal-logs keep-only-high-signal keeps lines matching level=error|warn.
	logLine := fmt.Sprintf("level=error %s", marker)
	podName := "otel-marker-" + strings.ToLower(uuid.DefaultGenerator.Generate()[:8])
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:    "logger",
					Image:   "busybox",
					Command: []string{"/bin/sh", "-c", fmt.Sprintf("while true; do echo %q; sleep 5; done", logLine)},
				},
			},
		},
	}
	_, err := clients.Kubernetes.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
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
