package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

func updatedObjects(ctx context.Context, nsfilter string) ([]string, error) {
	pods, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").List(ctx, metav1.ListOptions{
		LabelSelector: "app=aro-operator-master",
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("%d aro-operator-master pods found", len(pods.Items))
	}
	b, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw(ctx)
	if err != nil {
		return nil, err
	}

	rx := regexp.MustCompile(`msg="(Update|Create) ([-a-zA-Z/.]+)`)
	changes := rx.FindAllStringSubmatch(string(b), -1)
	result := make([]string, 0, len(changes))
	for _, change := range changes {
		if nsfilter == "" || strings.Contains(change[2], "/"+nsfilter+"/") {
			result = append(result, change[1]+" "+change[2])
		}
	}

	return result, nil
}

func dumpEvents(ctx context.Context, namespace string) error {
	events, err := clients.Kubernetes.EventsV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, event := range events.Items {
		log.Debugf("%s. %s. %s", event.Action, event.Reason, event.Note)
	}
	return nil
}

var _ = Describe("ARO Operator - Internet checking", func() {
	var originalURLs []string
	BeforeEach(func() {
		// save the originalURLs
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			Skip("skipping tests as aro-operator is not deployed")
		}

		Expect(err).NotTo(HaveOccurred())
		originalURLs = co.Spec.InternetChecker.URLs
	})
	AfterEach(func() {
		// set the URLs back again
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = originalURLs
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(context.Background(), co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())
	})
	Specify("the InternetReachable default list should all be reachable", func() {
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
	})

	Specify("the InternetReachable default list should all be reachable from worker", func() {
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
	})

	Specify("custom invalid site shows not InternetReachable", func() {
		// set an unreachable URL
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = []string{"https://localhost:1234/shouldnotexist"}
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(context.Background(), co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		// confirm the conditions are correct
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				log.Warn(err)
				return false, nil // swallow error
			}

			log.Debugf("ClusterStatus.Conditions %s", co.Status.Conditions)
			return conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster) &&
				conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker), nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Geneva Logging", func() {
	Specify("genevalogging must be repaired if deployment deleted", func() {
		mdsdReady := func() (bool, error) {
			done, err := ready.CheckDaemonSetIsReady(context.Background(), clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		if err != nil {
			// TODO: Remove dump once reason for flakes is clear
			err := dumpEvents(context.Background(), "openshift-azure-logging")
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(err).NotTo(HaveOccurred())

		initial, err := updatedObjects(context.Background(), "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())

		// delete the mdsd daemonset
		err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete(context.Background(), "mdsd", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// wait for it to be fixed
		err = wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		if err != nil {
			// TODO: Remove dump once reason for flakes is clear
			err := dumpEvents(context.Background(), "openshift-azure-logging")
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(err).NotTo(HaveOccurred())

		// confirm that only one object was updated
		final, err := updatedObjects(context.Background(), "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())
		if len(final)-len(initial) != 1 {
			log.Error("initial changes ", initial)
			log.Error("final changes ", final)
		}
		Expect(len(final) - len(initial)).To(Equal(1))
	})
})

var _ = Describe("ARO Operator - Cluster Monitoring ConfigMap", func() {
	Specify("cluster monitoring configmap should not have persistent volume config", func() {
		var cm *corev1.ConfigMap
		var err error
		configMapExists := func() (bool, error) {
			cm, err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		var configData monitoring.Config
		configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
		Expect(err).NotTo(HaveOccurred())

		err = codec.NewDecoderBytes(configDataJSON, &codec.JsonHandle{}).Decode(&configData)
		if err != nil {
			log.Warn(err)
		}

		Expect(configData.PrometheusK8s.Retention).To(BeEmpty())
		Expect(configData.PrometheusK8s.VolumeClaimTemplate).To(BeNil())
		Expect(configData.AlertManagerMain.VolumeClaimTemplate).To(BeNil())

	})

	Specify("cluster monitoring configmap should be restored if deleted", func() {
		configMapExists := func() (bool, error) {
			_, err := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Delete(context.Background(), "cluster-monitoring-config", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		_, err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - RBAC", func() {
	Specify("system:aro-sre ClusterRole should be restored if deleted", func() {
		clusterRoleExists := func() (bool, error) {
			_, err := clients.Kubernetes.RbacV1().ClusterRoles().Get(context.Background(), "system:aro-sre", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterRoleExists)
		Expect(err).NotTo(HaveOccurred())

		err = clients.Kubernetes.RbacV1().ClusterRoles().Delete(context.Background(), "system:aro-sre", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, clusterRoleExists)
		Expect(err).NotTo(HaveOccurred())

		_, err = clients.Kubernetes.RbacV1().ClusterRoles().Get(context.Background(), "system:aro-sre", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Conditions", func() {
	Specify("Cluster check conditions should not be failing", func() {
		clusterOperatorConditionsValid := func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			valid := true
			for _, condition := range arov1alpha1.ClusterChecksTypes() {
				if !conditions.IsTrue(co.Status.Conditions, condition) {
					valid = false
				}
			}
			return valid, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterOperatorConditionsValid)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = XDescribe("ARO Operator - MachineSet Controller", func() {
	Specify("operator should maintain at least two worker replicas", func() {
		ctx := context.Background()

		// TODO: MSFT Billing expects that we only scale a single node (4 VMs).
		// Need to work with billing pipeline to ensure we can run operator tests
		skipIfNotInDevelopmentEnv()

		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.Features.ReconcileMachineSet {
			Skip("MachineSet Controller is not enabled, skipping this test")
		}

		mss, err := clients.MachineAPI.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(mss.Items).NotTo(BeEmpty())

		// Zero all machinesets (avoid availability zone confusion)
		for _, object := range mss.Items {
			err = scale(object.Name, 0)
			Expect(err).NotTo(HaveOccurred())

			err = waitForScale(object.Name)
			Expect(err).NotTo(HaveOccurred())
		}

		// Re-count and assert that operator added back replicas
		modifiedMachineSets, err := clients.MachineAPI.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		replicaCount := 0
		for _, machineset := range modifiedMachineSets.Items {
			if machineset.Spec.Replicas != nil {
				replicaCount += int(*machineset.Spec.Replicas)
			}
		}
		Expect(replicaCount).To(BeEquivalentTo(minSupportedReplicas))

		// Restore previous state
		for _, ms := range mss.Items {
			err := scale(ms.Name, *ms.Spec.Replicas)
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("ARO Operator - ImageConfig Controller", func() {
	FSpecify("Operator should add appropriate registries", func() {
		dockerPodSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "docker-alpine-pod",
				Labels: map[string]string{"imgcfg": "e2e"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "alpine",
						Image: "alpine",
					},
				},
				RestartPolicy: "Never",
			},
		}
		quayPodSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "quay-alpine-pod",
				Labels: map[string]string{"imgcfg": "e2e"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "alpine",
						Image: "quay.io/kmagdani/alpine",
					},
				},
				RestartPolicy: "Never",
			},
		}
		ctx := context.Background()

		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.Features.ReconcileImageConfig {
			Skip("ImageConfig Controller is not enabled, skipping this test")
		}
		// ! Get Imageconfig object
		imageconfig, err := clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		// ! Add quay.io to allowed Registries
		imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, "quay.io")
		// ! Update Registries
		_, err = clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		// ! Create Alpine pod from docker.io
		_, err = clients.Kubernetes.CoreV1().Pods("default").Create(ctx, dockerPodSpec, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// ! Create Alpine pod from quay.io
		_, err = clients.Kubernetes.CoreV1().Pods("default").Create(ctx, quayPodSpec, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		// time.Sleep(60 * time.Second)
		// ! Watch pod status
		watch, err := clients.Kubernetes.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		go func() {
			for event := range watch.ResultChan() {
				// fmt.Printf("Type: %v\n", event.Type)
				p, ok := event.Object.(*v1.Pod)
				// if p ==
				if !ok {
					log.Fatal("unexpected type")
				}
				// fmt.Println(p.Status.ContainerStatuses)
				fmt.Println(p.Name + "-" + string(p.Status.Phase))
			}
		}()
		time.Sleep(60 * time.Second)

	})
})
