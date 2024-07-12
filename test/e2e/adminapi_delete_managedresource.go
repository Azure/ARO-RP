package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var loadBalancerService = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test",
	},
	Spec: corev1.ServiceSpec{
		Type: corev1.ServiceTypeLoadBalancer,
		Ports: []corev1.ServicePort{
			{
				Name:     "service-443",
				Protocol: corev1.ProtocolTCP,
				Port:     int32(443),
			},
		},
	},
}

var _ = Describe("[Admin API] Delete managed resource action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should be possible to delete managed cluster resources", func(ctx context.Context) {
		var service *corev1.Service
		var lbRuleID string
		var fipConfigID string
		var pipAddressID string

		const namespace = "default"

		By("creating a test service of type loadbalancer")
		creationFunc := clients.Kubernetes.CoreV1().Services(namespace).Create
		CreateK8sObjectWithRetry(ctx, creationFunc, &loadBalancerService, metav1.CreateOptions{})

		defer func() {
			By("cleaning up the k8s loadbalancer service")
			CleanupK8sResource[*corev1.Service](
				ctx, clients.Kubernetes.CoreV1().Services(namespace), loadBalancerService.Name,
			)
		}()

		// wait for ingress IP to be assigned as this indicate the service is ready
		Eventually(func(g Gomega, ctx context.Context) {
			getFunc := clients.Kubernetes.CoreV1().Services(namespace).Get
			service = GetK8sObjectWithRetry(ctx, getFunc, loadBalancerService.Name, metav1.GetOptions{})
			g.Expect(service.Status.LoadBalancer.Ingress).To(HaveLen(1))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("getting the newly created k8s service frontend IP configuration")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		rgName := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')
		lbName, err := getInfraID(ctx)
		Expect(err).NotTo(HaveOccurred())

		lb, err := clients.LoadBalancers.Get(ctx, rgName, lbName, "")
		Expect(err).NotTo(HaveOccurred())

		for _, fipConfig := range *lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations {
			if !strings.Contains(*fipConfig.PublicIPAddress.ID, "default-v4") && !strings.Contains(*fipConfig.PublicIPAddress.ID, "pip-v4") {
				lbRuleID = *(*fipConfig.LoadBalancingRules)[0].ID
				fipConfigID = *fipConfig.ID
				pipAddressID = *fipConfig.PublicIPAddress.ID
			}
		}

		By("deleting the associated loadbalancer rule")
		testDeleteManagedResourceOK(ctx, lbRuleID)

		By("deleting the associated frontend ip config")
		testDeleteManagedResourceOK(ctx, fipConfigID)

		By("deleting the associated public ip address")
		testDeleteManagedResourceOK(ctx, pipAddressID)
	})

	It("should NOT be possible to delete a resource not within the cluster's managed resource group", func(ctx context.Context) {
		By("trying to delete the master subnet")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{*oc.OpenShiftClusterProperties.MasterProfile.SubnetID}}, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("should NOT be possible to delete the private link service in the cluster's managed resource group", func(ctx context.Context) {
		By("trying to delete the private link service")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		// Fake name prevents accidentally deleting the PLS but still validates guardrail logic works.
		plsResourceID := fmt.Sprintf("%s/providers/Microsoft.Network/PrivateLinkServices/%s", *oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, "fake-pls")

		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{plsResourceID}}, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})
})

func testDeleteManagedResourceOK(ctx context.Context, resourceID string) {
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/deletemanagedresource", url.Values{"managedResourceID": []string{resourceID}}, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}
