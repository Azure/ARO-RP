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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
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
		Expect(co.Status.Conditions.IsTrueFor(arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
	})

	Specify("the InternetReachable default list should all be reachable from worker", func() {
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(co.Status.Conditions.IsTrueFor(arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
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
			return co.Status.Conditions.IsFalseFor(arov1alpha1.InternetReachableFromMaster) &&
				co.Status.Conditions.IsFalseFor(arov1alpha1.InternetReachableFromWorker), nil
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

var _ = Describe("ARO Operator - Routefix Daemonset", func() {
	// remove this once the change where the operator manages the routefix
	// daemonset is in production
	Specify("routefix DaemonSet must be Ready", func() {
		dsReady := func() (bool, error) {
			done, err := ready.CheckDaemonSetIsReady(context.Background(), clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix"), "routefix")()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, dsReady)
		Expect(err).NotTo(HaveOccurred())

	})
	Specify("routefix Pods must all be Running before testing DaemonSet deletion", func() {
		dsReady := func() (bool, error) {
			clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix")
			done, err := ready.CheckPodsAreRunning(context.Background(), clients.Kubernetes.CoreV1().Pods("openshift-azure-routefix"), map[string]string{"app": "routefix"})()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, dsReady)
		Expect(err).NotTo(HaveOccurred())

	})
	Specify("routefix must be repaired if DaemonSet is deleted", func() {
		dsReady := func() (bool, error) {
			done, err := ready.CheckDaemonSetIsReady(context.Background(), clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix"), "routefix")()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, dsReady)
		Expect(err).NotTo(HaveOccurred())

		// delete the routefix daemonset
		err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix").Delete(context.Background(), "routefix", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// wait for it to be fixed
		err = wait.PollImmediate(30*time.Second, 15*time.Minute, dsReady)
		Expect(err).NotTo(HaveOccurred())

		_, err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix").Get(context.Background(), "routefix", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
	Specify("routefix Pods must all be Running after testing DaemonSet deletion", func() {
		dsReady := func() (bool, error) {
			clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-routefix")
			done, err := ready.CheckPodsAreRunning(context.Background(), clients.Kubernetes.CoreV1().Pods("openshift-azure-routefix"), map[string]string{"app": "routefix"})()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, dsReady)
		Expect(err).NotTo(HaveOccurred())
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
				if !co.Status.Conditions.IsTrueFor(condition) {
					valid = false
				}
			}
			return valid, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterOperatorConditionsValid)
		Expect(err).NotTo(HaveOccurred())
	})
})
