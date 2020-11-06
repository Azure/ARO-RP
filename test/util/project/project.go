package project

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	projectv1 "github.com/openshift/api/project/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Project struct {
	projectV1Client projectv1client.ProjectV1Interface
	cli             kubernetes.Interface
	name            string
}

func NewProject(cli kubernetes.Interface, projectV1Client projectv1client.ProjectV1Interface, name string) Project {
	return Project{
		projectV1Client: projectV1Client,
		cli:             cli,
		name:            name,
	}
}

func (p Project) Create(ctx context.Context) error {
	_, err := p.projectV1Client.Projects().Create(ctx, &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.name,
		},
	}, metav1.CreateOptions{})
	return err
}

func (p Project) Delete(ctx context.Context) error {
	return p.projectV1Client.Projects().Delete(ctx, p.name, metav1.DeleteOptions{})
}

// VerifyProjectIsReady verifies that the project and relevant resources have been created correctly and returns error otherwise
func (p Project) Verify(ctx context.Context) error {
	_, err := p.cli.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx,
		&authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: p.name,
					Verb:      "create",
					Resource:  "pods",
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	sa, err := p.cli.CoreV1().ServiceAccounts(p.name).Get(ctx, "default", metav1.GetOptions{})
	if err != nil || errors.IsNotFound(err) {
		return fmt.Errorf("Error retrieving default ServiceAccount")
	}

	if len(sa.Secrets) == 0 {
		return fmt.Errorf("Default ServiceAccount does not have secrets")
	}

	proj, err := p.projectV1Client.Projects().Get(ctx, p.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, found := proj.Annotations["openshift.io/sa.scc.uid-range"]
	if !found {
		return fmt.Errorf("SCC annotation does not exist")
	}
	return nil
}

// VerifyProjectIsDeleted verifies that the project does not exist and returns error if a project exists
// or if it encounters an error other than NotFound
func (p Project) VerifyProjectIsDeleted(ctx context.Context) error {
	_, err := p.projectV1Client.Projects().Get(ctx, p.name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	return fmt.Errorf("Project exists")
}
