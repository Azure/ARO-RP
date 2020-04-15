package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset/installconfig"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func (i *Installer) ensureCloudCredentialConfiguration(ctx context.Context, platformCreds *installconfig.PlatformCreds) error {
	err := ensureRole(ctx, i.kubernetescli)
	if err != nil {
		return err
	}
	err = ensureRoleBinding(ctx, i.kubernetescli)
	if err != nil {
		return err
	}
	return ensureSecret(ctx, i.kubernetescli, platformCreds)
}

func ensureRoleBinding(ctx context.Context, cli kubernetes.Interface) error {
	name := "aro-cloud-provider-secret-read"
	namespace := "kube-system"
	rb, err := cli.RbacV1().RoleBindings(namespace).Create(&rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kube-system",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "aro-cloud-provider-secret-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "azure-cloud-provider",
				Namespace: "kube-system",
			},
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_rb, err := cli.RbacV1().RoleBindings(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		rb.ResourceVersion = _rb.ResourceVersion
		_, err = cli.RbacV1().RoleBindings(namespace).Update(rb)
		return err
	})
}

func ensureRole(ctx context.Context, cli kubernetes.Interface) error {
	name := "aro-cloud-provider-secret-reader"
	namespace := "kube-system"
	r, err := cli.RbacV1().Roles(namespace).Create(&rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kube-system",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{"azure-cloud-provider"},
				Verbs:         []string{"get"},
			},
		},
	})
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_r, err := cli.RbacV1().Roles(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		r.ResourceVersion = _r.ResourceVersion
		_, err = cli.RbacV1().Roles(namespace).Update(r)
		return err
	})
}

func ensureSecret(ctx context.Context, cli kubernetes.Interface, platformCreds *installconfig.PlatformCreds) error {
	name := "azure-cloud-provider"
	namespace := "kube-system"
	secret := &v1.Secret{
		Type: v1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	// config is used to created compatible secret to trigger azure cloud
	// controller config merge behaviour
	// https://github.com/openshift/origin/blob/release-4.3/vendor/k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/azure/azure_config.go#L82
	config := struct {
		AADClientID     string `json:"aadClientId" yaml:"aadClientId"`
		AADClientSecret string `json:"aadClientSecret" yaml:"aadClientSecret"`
	}{
		AADClientID:     platformCreds.Azure.ClientID,
		AADClientSecret: platformCreds.Azure.ClientSecret,
	}
	secretData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	secret.Data = map[string][]byte{
		"cloud-config": secretData,
	}

	_, err = cli.CoreV1().Secrets(namespace).Create(secret)
	if !errors.IsAlreadyExists(err) {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_secret, err := cli.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		secret.ResourceVersion = _secret.ResourceVersion
		_, err = cli.CoreV1().Secrets(namespace).Update(secret)
		return err
	})
}
