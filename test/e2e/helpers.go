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

// This function takes a get function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func GetK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, get K8sGetFunc[T], name string, options metav1.GetOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := get(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function gets the logs for the specified pod in the named namespace. It gets them with some
// retry logic and returns the raw string-ified body after asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
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

// This function takes a list function like clients.Kubernetes.CoreV1().Nodes().List and the
// parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func ListK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, list K8sListFunc[T], options metav1.ListOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := list(ctx, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a create function like clients.Kubernetes.CoreV1().Pods(namespace).Create
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func CreateK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, create K8sCreateFunc[T], obj T, options metav1.CreateOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := create(ctx, obj, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a delete function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete
// and the parameters for it. It then makes the call with some retry logic.
//
// By default the call is retried for 5s with a 250ms interval.
func DeleteK8sObjectWithRetry(
	ctx context.Context, delete K8sDeleteFunc, name string, options metav1.DeleteOptions,
) {
	Eventually(func(g Gomega, ctx context.Context) {
		err := delete(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
}

type Project struct {
	projectClient projectclient.Interface
	cli           kubernetes.Interface
	Name          string
}

func NewProject(
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

// VerifyProjectIsReady verifies that the project and relevant resources have been created correctly and returns error otherwise
func (p Project) Verify(ctx context.Context) error {
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
	sa := GetK8sObjectWithRetry(
		ctx, p.cli.CoreV1().ServiceAccounts(p.Name).Get, "default", metav1.GetOptions{},
	)

	if len(sa.Secrets) == 0 {
		return fmt.Errorf("default ServiceAccount does not have secrets")
	}

	project := GetK8sObjectWithRetry(
		ctx, p.projectClient.ProjectV1().Projects().Get, p.Name, metav1.GetOptions{},
	)
	_, found := project.Annotations["openshift.io/sa.scc.uid-range"]
	if !found {
		return fmt.Errorf("SCC annotation does not exist")
	}
	return nil
}

// VerifyProjectIsDeleted verifies that the project does not exist and returns error if a project exists
// or if it encounters an error other than NotFound
func (p Project) VerifyProjectIsDeleted(ctx context.Context) error {
	_, err := p.projectClient.ProjectV1().Projects().Get(ctx, p.Name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil
	}

	return fmt.Errorf("Project exists")
}
