package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	projectv1 "github.com/openshift/api/project/v1"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

var (
	DefaultTimeout  = 10 * time.Minute
	PollingInterval = 250 * time.Millisecond
)

type K8sGetFunc[T kruntime.Object] func(ctx context.Context, name string, options metav1.GetOptions) (T, error)
type K8sListFunc[T kruntime.Object] func(ctx context.Context, options metav1.ListOptions) (T, error)
type K8sCreateFunc[T kruntime.Object] func(ctx context.Context, object T, options metav1.CreateOptions) (T, error)
type K8sDeleteFunc func(ctx context.Context, name string, options metav1.DeleteOptions) error

// GetK8sObjectWithRetry takes a get function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
func GetK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, getFunc K8sGetFunc[T], name string, options metav1.GetOptions,
) (result T, err error) {
	Eventually(func(g Gomega, ctx context.Context) {
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return result, err
}

// GetK8sPodLogsWithRetry gets the logs for the specified pod in the named namespace. It gets them with some
// retry logic and returns the raw string-ified body after asserting there were no errors.
func GetK8sPodLogsWithRetry(
	ctx context.Context, namespace string, name string, options corev1.PodLogOptions,
) (rawBody string) {
	Eventually(func(g Gomega, ctx context.Context) {
		body, err := clients.Kubernetes.CoreV1().Pods(namespace).GetLogs(name, &options).DoRaw(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		rawBody = string(body)
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return
}

// ListK8sObjectWithRetry takes a list function like clients.Kubernetes.CoreV1().Nodes().List and the
// parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
func ListK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, listFunc K8sListFunc[T], options metav1.ListOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := listFunc(ctx, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// CreateK8sObjectWithRetry takes a create function like clients.Kubernetes.CoreV1().Pods(namespace).Create
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
func CreateK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, createFunc K8sCreateFunc[T], obj T, options metav1.CreateOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := createFunc(ctx, obj, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// DeleteK8sObjectWithRetry takes a delete function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete
// and the parameters for it. It then makes the call with some retry logic.
func DeleteK8sObjectWithRetry(
	ctx context.Context, deleteFunc K8sDeleteFunc, name string, options metav1.DeleteOptions,
) (err error) {
	Eventually(func(g Gomega, ctx context.Context) {
		err := deleteFunc(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return
}

type AllowedCleanUpAPIInterface[T kruntime.Object] interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
	Delete(ctx context.Context, name string, options metav1.DeleteOptions) error
}

// CleanupK8sResource takes a client that knows how to issue a GET and DELETE call for a given resource.
// It then issues a delete request then and polls the API to until the resource is no longer found.
//
// Note: If the DELETE request receives a 404 we assume the resource has been cleaned up successfully.
func CleanupK8sResource[T kruntime.Object](
	ctx context.Context, client AllowedCleanUpAPIInterface[T], name string,
) {
	DefaultEventuallyTimeout = 10 * time.Minute
	PollingInterval = 1 * time.Second
	err := DeleteK8sObjectWithRetry(
		ctx, client.Delete, name, metav1.DeleteOptions{},
	)

	if err != nil && kerrors.IsNotFound(err) {
		return
	}

	Eventually(func(g Gomega, ctx context.Context) {
		_, err := GetK8sObjectWithRetry(ctx, client.Get, name, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
	}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
}

type Project struct {
	projectClient projectclient.Interface
	cli           kubernetes.Interface
	Name          string
}

func BuildNewProject(
	ctx context.Context, cli kubernetes.Interface, projectClient projectclient.Interface, name string,
) Project {
	p := Project{
		projectClient: projectClient,
		cli:           cli,
		Name:          name,
	}
	createFunc := p.projectClient.ProjectV1().Projects().Create
	CreateK8sObjectWithRetry(ctx, createFunc, &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Name,
		},
	}, metav1.CreateOptions{})
	return p
}

func (p Project) CleanUp(ctx context.Context) {
	p.Delete(ctx)
	p.VerifyProjectIsDeleted(ctx)
}

func (p Project) Delete(ctx context.Context) {
	deleteFunc := p.projectClient.ProjectV1().Projects().Delete
	DeleteK8sObjectWithRetry(ctx, deleteFunc, p.Name, metav1.DeleteOptions{})
}

// VerifyProjectIsReady verifies that the project and relevant resources have been created correctly
// and returns an error if any.
func (p Project) VerifyProjectIsReady(ctx context.Context) error {
	By("creating a SelfSubjectAccessReviews")
	CreateK8sObjectWithRetry(ctx, p.cli.AuthorizationV1().SelfSubjectAccessReviews().Create,
		&authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: p.Name,
					Verb:      "create",
					Resource:  "pods",
				},
			},
		}, metav1.CreateOptions{})

	By("getting the relevant SA")
	sa, _ := GetK8sObjectWithRetry(
		ctx, p.cli.CoreV1().ServiceAccounts(p.Name).Get, "default", metav1.GetOptions{},
	)

	if len(sa.Secrets) == 0 {
		return fmt.Errorf("default ServiceAccount does not have secrets")
	}

	project, _ := GetK8sObjectWithRetry(
		ctx, p.projectClient.ProjectV1().Projects().Get, p.Name, metav1.GetOptions{},
	)
	_, found := project.Annotations["openshift.io/sa.scc.uid-range"]
	if !found {
		return fmt.Errorf("SCC annotation does not exist")
	}
	return nil
}

// VerifyProjectIsDeleted verifies that the project does not exist by polling it.
func (p Project) VerifyProjectIsDeleted(ctx context.Context) {
	Eventually(func(g Gomega, ctx context.Context) {
		_, err := p.projectClient.ProjectV1().Projects().Get(ctx, p.Name, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
	}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
}
