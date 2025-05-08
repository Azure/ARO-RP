package serversideapply

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// The current kubernetes testing client does not propery handle Apply actions so we reimplement it here.
// See https://github.com/kubernetes/client-go/issues/1184 for more details.
func CliWithApply(objectTypes []string, object ...runtime.Object) *fake.Clientset {
	// This slice will have to be kept up-to-date with the switch statement
	// further down inside the for loop...
	supportedObjectTypes := []string{
		"secrets",
		"deployments",
	}

	fc := fake.NewSimpleClientset(object...)
	for _, ot := range objectTypes {
		foundOt := false
		for _, sot := range supportedObjectTypes {
			if ot == sot {
				foundOt = true
				break
			}
		}

		if !foundOt {
			panic(fmt.Sprintf("Kubernetes object type %s needs to be added to ARO-RP/test/util/serversideapply's CliWithApply (see doc comment on function for context)", ot))
		}

		fc.PrependReactor("patch", ot, func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			pa := action.(ktesting.PatchAction)
			if pa.GetPatchType() == types.ApplyPatchType {
				// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
				// if an apply patch occurs for a secret that doesn't yet exist, create it.
				// However, we already hold the fakeclient lock, so we can't use the front door.
				rfunc := ktesting.ObjectReaction(fc.Tracker())
				_, obj, err := rfunc(
					ktesting.NewGetAction(pa.GetResource(), pa.GetNamespace(), pa.GetName()),
				)

				var blankObject runtime.Object
				switch ot {
				case "secrets":
					blankObject = &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pa.GetName(),
							Namespace: pa.GetNamespace(),
						},
					}
				case "deployments":
					blankObject = &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pa.GetName(),
							Namespace: pa.GetNamespace(),
						},
					}
				}

				if kerrors.IsNotFound(err) || obj == nil {
					_, _, _ = rfunc(
						ktesting.NewCreateAction(
							pa.GetResource(),
							pa.GetNamespace(),
							blankObject,
						),
					)
				}
				return rfunc(ktesting.NewPatchAction(
					pa.GetResource(),
					pa.GetNamespace(),
					pa.GetName(),
					types.StrategicMergePatchType,
					pa.GetPatch()))
			}
			return false, nil, nil
		},
		)
	}
	return fc
}
