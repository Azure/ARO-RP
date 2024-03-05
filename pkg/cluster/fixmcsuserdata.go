package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type userData struct {
	api.MissingFields
	Ignition struct {
		api.MissingFields
		Config struct {
			api.MissingFields
			Merge []struct { // ignition 3.x
				api.MissingFields
				Source string `json:"source,omitempty"`
			} `json:"merge,omitempty"`
			Append []struct { // ignition 2.x
				api.MissingFields
				Source string `json:"source,omitempty"`
			} `json:"append,omitempty"`
		} `json:"config,omitempty"`
	} `json:"ignition,omitempty"`
}

func (m *manager) enumerateUserDataSecrets(ctx context.Context) map[corev1.SecretReference]struct{} {
	secretRefs := map[corev1.SecretReference]struct{}{}

	machinesets, err := m.maocli.MachineV1beta1().MachineSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
	} else {
		for _, machineset := range machinesets.Items {
			ref, err := getUserDataSecretReference(&machineset.ObjectMeta, &machineset.Spec.Template.Spec)
			if err != nil {
				m.log.Printf("%s/%s: %s", machineset.Namespace, machineset.Name, err)
				continue
			}
			if ref != nil {
				secretRefs[*ref] = struct{}{}
			}
		}
	}

	machines, err := m.maocli.MachineV1beta1().Machines("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.Print(err)
	} else {
		for _, machine := range machines.Items {
			ref, err := getUserDataSecretReference(&machine.ObjectMeta, &machine.Spec)
			if err != nil {
				m.log.Printf("%s/%s: %s", machine.Namespace, machine.Name, err)
				continue
			}
			if ref != nil {
				secretRefs[*ref] = struct{}{}
			}
		}
	}

	return secretRefs
}

func getUserDataSecretReference(objMeta *metav1.ObjectMeta, spec *machinev1beta1.MachineSpec) (*corev1.SecretReference, error) {
	if spec.ProviderSpec.Value == nil || objMeta == nil {
		return nil, nil
	}

	obj := &unstructured.Unstructured{}
	err := obj.UnmarshalJSON(spec.ProviderSpec.Value.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling provider spec: %w", err)
	}

	machineProviderSpec := obj.UnstructuredContent()

	secret := &corev1.SecretReference{}

	userDataSecretNamespace, ok, err := unstructured.NestedString(machineProviderSpec, "userDataSecret", "namespace")
	if err != nil {
		return nil, fmt.Errorf("failed getting secret reference: %w", err)
	} else if !ok || userDataSecretNamespace == "" {
		secret.Namespace = objMeta.Namespace
	} else {
		secret.Namespace = userDataSecretNamespace
	}

	userDataSecretName, ok, err := unstructured.NestedString(machineProviderSpec, "userDataSecret", "name")
	if err != nil {
		return nil, fmt.Errorf("failed getting secret reference name: %w", err)
	} else if !ok || userDataSecretName == "" {
		return nil, nil
	} else {
		secret.Name = userDataSecretName
	}

	return secret, nil
}

func (m *manager) fixMCSUserData(ctx context.Context) error {
	h := codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			EncodeOptions: codec.EncodeOptions{
				Canonical: true,
			},
		},
	}

	for secretRef := range m.enumerateUserDataSecrets(ctx) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			s, err := m.kubernetescli.CoreV1().Secrets(secretRef.Namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			var userData *userData
			err = codec.NewDecoderBytes(s.Data["userData"], &h).Decode(&userData)
			if err != nil {
				return fmt.Errorf("failed decoding secret userData: %w", err)
			}

			var changed bool
			for i, a := range userData.Ignition.Config.Merge {
				var _changed bool
				a.Source, _changed, err = m.fixSource(a.Source)
				if err != nil {
					return fmt.Errorf("failed fixing domain source: %w", err)
				}

				changed = changed || _changed

				userData.Ignition.Config.Merge[i] = a
			}

			for i, a := range userData.Ignition.Config.Append {
				var _changed bool
				a.Source, _changed, err = m.fixSource(a.Source)
				if err != nil {
					return fmt.Errorf("failed fixing domain source: %w", err)
				}

				changed = changed || _changed

				userData.Ignition.Config.Append[i] = a
			}

			if !changed {
				return nil
			}

			var b []byte
			err = codec.NewEncoderBytes(&b, &h).Encode(userData)
			if err != nil {
				return fmt.Errorf("failed encoding userData: %w", err)
			}

			s.Data["userData"] = b

			_, err = m.kubernetescli.CoreV1().Secrets(secretRef.Namespace).Update(ctx, s, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			m.log.Printf("%s/%s: %s", secretRef.Namespace, secretRef.Name, err)
		}
	}

	return nil
}

func (m *manager) fixSource(source string) (string, bool, error) {
	intIP := net.ParseIP(m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP)

	domain := m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + m.env.Domain()
	}

	u, err := url.Parse(source)
	if err != nil {
		return "", false, err
	}

	var changed bool
	if u.Hostname() == "api-int."+domain {
		u.Host = intIP.String() + ":" + u.Port()
		changed = true
	}

	return u.String(), changed, nil
}
