package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	version = "4.15.30"
	tempDir = path.Join("/tmp/credreqs", version)
)

func main() {
		image := fmt.Sprintf("quay.io/openshift-release-dev/ocp-release:%s-x86_64", version)
		err := exec.Command("./oc", "adm", "release", "extract", "--credentials-requests", "--to", tempDir, image).Run()
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func(ctx context.Context) {
			err = os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	DescribeTable("should cover required permissions", func(ctx context.Context, credReqFile, roleDefinitionId string) {
		By("Reading the credentials request")
		f, err := os.Open(path.Join(tempDir, credReqFile))
		Expect(err).NotTo(HaveOccurred())

		credreq, err := GetAzureCredentialsRequest(f)
		Expect(err).NotTo(HaveOccurred())

		By("Fetching the Azure role definition")
		var spec cloudcredentialv1.AzureProviderSpec
		err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &spec)
		Expect(err).NotTo(HaveOccurred())

		role, err := clients.RoleDefinitions.Get(ctx, "", roleDefinitionId)
		Expect(err).NotTo(HaveOccurred())
		// It's not guaranteed to have only one permission, but it's a good enough assumption for now.
		// Expect here to make sure the test fails if the assumption is wrong.
		Expect(*role.Permissions).To(HaveLen(1))
		permission := (*role.Permissions)[0]

		By("Comparing the permissions")
		Expect(err).NotTo(HaveOccurred())
		Expect(*permission.Actions).To(ContainElements(spec.Permissions))
		Expect(*permission.DataActions).To(ContainElements(spec.DataPermissions))
	},
		Entry(
			"Cloud Controller Manager",
			"0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml",
			"a1f96423-95ce-4224-ab27-4e3dc72facd4",
		),
		Entry(
			"Machine API Operator",
			"0000_30_machine-api-operator_00_credentials-request.yaml",
			"0358943c-7e01-48ba-8889-02cc51d78637",
		),
		Entry(
			"Cluster Image Registry Operator",
			"0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml",
			"8b32b316-c2f5-4ddf-b05b-83dacd2d08b5",
		),
		Entry(
			"Cluster Ingress Operator",
			"0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml",
			"0336e1d3-7a87-462b-b6db-342b63f7802c",
		),
		Entry(
			"Cluster Network Operator",
			"0000_50_cluster-network-operator_02-cncc-credentials.yaml",
			"be7a6435-15ae-4171-8f30-4a343eff9e8f",
		),
		Entry(
			"Cluster Storage Operator",
			"0000_50_cluster-storage-operator_03_credentials_request_azure.yaml",
			"5b7237c5-45e1-49d6-bc18-a1f62f400748",
		),
		Entry(
			"Cluster Storage Operator (Azure File)",
			"0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml",
			"0d7aedc0-15fd-4a67-a412-efad370c947e",
		),
	)
})
}

func GetAzureCredentialsRequest(r io.Reader) (*cloudcredentialv1.CredentialsRequest, error) {
	dec := yaml.NewYAMLOrJSONDecoder(r, 4096)
	var credreq cloudcredentialv1.CredentialsRequest
	for {
		err := dec.Decode(&credreq)
		if errors.Is(err, io.EOF) {
			return nil, errors.New("azure credentials request not found")
		}
		if err != nil {
			return nil, err
		}
		unknown := runtime.Unknown{}
		err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &unknown)
		if err != nil {
			return nil, err
		}
		By(fmt.Sprintf("Found credentials request %s", unknown.Kind))
		if unknown.Kind == "AzureProviderSpec" {
			return &credreq, nil
		}
	}
}
