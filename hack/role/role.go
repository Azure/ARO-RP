package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"

	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type permissionSet struct {
	name             string
	manifest         string
	roleDefinitionId string
}

type manager struct {
	roleDefinitions armauthorization.RoleDefinitionsClient
}

type response struct {
	Tags          []tag `json:"tags"`
	Page          int   `json:"page"`
	HasAdditional bool  `json:"has_additional"`
}

type tag struct {
	Name         string   `json:"name"`
	LastModified datetime `json:"last_modified"`
}

type datetime struct {
	time.Time
}

func (d *datetime) UnmarshalJSON(data []byte) error {
	dd, err := time.Parse(fmt.Sprintf("\"%s\"", time.RFC1123Z), string(data))
	if err != nil {
		return err
	}
	d.Time = dd
	return nil
}

type missingPermissionsError struct{}

func (m *missingPermissionsError) Error() string {
	return "missing permissions"
}

type emptyVersionsError struct{}

func (e *emptyVersionsError) Error() string {
	return "no versions found"
}

var rxImage = regexp.MustCompile(`\d+\.\d+\.\d+(-(rc|ec).\d+)?-x86_64`)

var (
	verifiedVersion = flag.String("verified-version", "", "verified version")
	targetVersion   = flag.String("target-version", "", "(optional) version to be verified")
	ocBinary        = flag.String("oc-bin", "oc", "path to oc binary")
	permissionSets  = []permissionSet{
		{
			"cloud-controller-manager",
			"0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml",
			"a1f96423-95ce-4224-ab27-4e3dc72facd4",
		},
		{
			"machine-api",
			"0000_30_machine-api-operator_00_credentials-request.yaml",
			"0358943c-7e01-48ba-8889-02cc51d78637",
		},
		{
			"image-registry",
			"0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml",
			"8b32b316-c2f5-4ddf-b05b-83dacd2d08b5",
		},
		{
			"ingress",
			"0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml",
			"0336e1d3-7a87-462b-b6db-342b63f7802c",
		},
		{
			"cloud-network-config",
			"0000_50_cluster-network-operator_02-cncc-credentials.yaml",
			"be7a6435-15ae-4171-8f30-4a343eff9e8f",
		},
		{
			"disk-csi-driver",
			"0000_50_cluster-storage-operator_03_credentials_request_azure.yaml",
			"5b7237c5-45e1-49d6-bc18-a1f62f400748",
		},
		{
			"file-csi-driver",
			"0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml",
			"0d7aedc0-15fd-4a67-a412-efad370c947e",
		},
	}
)

func main() {
	flag.Parse()
	if *verifiedVersion == "" {
		panic("verified-version is required")
	}

	ctx := context.Background()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	log := logrus.NewEntry(logger)
	environment, err := env.NewCoreForCI(ctx, log, env.SERVICE_TOOLING) // we don't use log here
	if err != nil {
		panic(err)
	}

	options := environment.Environment().EnvironmentCredentialOptions()
	tokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		panic(err)
	}
	roleDefinitions, err := armauthorization.NewArmRoleDefinitionsClient(tokenCredential, environment.SubscriptionID(), nil)
	if err != nil {
		panic(err)
	}

	m := manager{
		roleDefinitions: roleDefinitions,
	}

	verifiedDir, err := os.MkdirTemp("", *verifiedVersion)
	if err != nil {
		panic(err)
	}
	err = extractCredReq(*verifiedVersion, verifiedDir)
	if err != nil {
		panic(err)
	}

	var vers []string
	if *targetVersion != "" {
		// if target version is specified, validate only that version
		vers = []string{*targetVersion}
	} else {
		// validate all available versions
		vers, err = versionsToValidate(*verifiedVersion)
		if err != nil {
			panic(err)
		}
	}

	var missing []string
	for _, v := range vers {
		fmt.Println("Checking", v)
		targetDir, err := os.MkdirTemp("", v)
		if err != nil {
			panic(err)
		}
		err = extractCredReq(v, targetDir)
		if err != nil {
			panic(err)
		}

		if err = validate(ctx, &m, verifiedDir, targetDir, permissionSets); err != nil {
			// To check all versions, we need to continue even if there are missing permissions.
			if errors.Is(err, &missingPermissionsError{}) {
				missing = append(missing, v)
				continue
			}
			panic(err)
		}
	}
	if len(missing) > 0 {
		for _, v := range missing {
			fmt.Printf("Permissions are missing in %s\n", v)
		}
		os.Exit(1)
	}
}

// versionsToValidate returns all the latest patch versions for each minor version later than the given verified version.
func versionsToValidate(verifiedVersion string) ([]string, error) {
	v, err := version.ParseVersion(verifiedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	var vers []string
	for ; ; v.V[1]++ {
		latest, err := getLatestVersion(v)
		if err != nil {
			if errors.Is(err, &emptyVersionsError{}) {
				break
			}
			return nil, fmt.Errorf("failed to get latest image for %s: %w", v, err)
		}
		vers = append(vers, latest)
	}
	return vers, nil
}

// getLatestVersion returns the latest available patch version for the given minor version.
// The version could be a candidate version (e.g. 4.17.0-rc1, 4.17.0-ec2).
func getLatestVersion(v *version.Version) (string, error) {
	tags, err := getAvailableTags(v)
	if err != nil {
		return "", fmt.Errorf("failed to get available tags: %w", err)
	}
	if len(tags) == 0 {
		return "", &emptyVersionsError{}
	}
	t := slices.MaxFunc(tags, func(a, b tag) int {
		return a.LastModified.Compare(b.LastModified.Time)
	})
	return strings.TrimSuffix(t.Name, "-x86_64"), nil
}

// getAvailableTags returns all available tags in quay.io/openshift-release-dev/ocp-release for given minor version.
// The tags are like 4.14.37-x86_64, 4.17.0-rc1-x86_64, 4.17.0-ec2-x86_64, etc.
func getAvailableTags(v *version.Version) ([]tag, error) {
	hasAdditional := true
	page := 1
	var tags []tag
	for hasAdditional {
		// https://docs.redhat.com/en/documentation/red_hat_quay/3/html-single/red_hat_quay_api_guide/index#red_hat_quay_application_programming_interface_api
		// It paginates with hasAdditional and page parameters to fetch all tags.
		res, err := http.Get(fmt.Sprintf("https://quay.io/api/v1/repository/openshift-release-dev/ocp-release/tag/?limit=100&page=%d&filter_tag_name=like:%s.%%-x86_64", page, v.MinorVersion()))
		if err != nil {
			return nil, fmt.Errorf("failed to list tags: %w", err)
		}
		var r response
		if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode tag: %w", err)
		}

		for _, t := range r.Tags {
			if !rxImage.MatchString(t.Name) {
				continue
			}
			tags = append(tags, t)
		}
		hasAdditional = r.HasAdditional
		page = r.Page + 1
	}
	return tags, nil
}

func extractCredReq(v, outDir string) error {
	fmt.Println("Extracting", v)
	image := fmt.Sprintf("quay.io/openshift-release-dev/ocp-release:%s-x86_64", v)
	err := exec.Command(*ocBinary, "adm", "release", "extract", "--credentials-requests", "--to", outDir, image).Run()
	if err != nil {
		return fmt.Errorf("failed to extract credentials requests: %w", err)
	}
	return nil
}

func validate(ctx context.Context, m *manager, verifiedDir, targetDir string, permissionSets []permissionSet) error {
	fmt.Println("Validating permissions")
	missing := false

	for _, ps := range permissionSets {
		verifiedSpec, err := m.GetAzureProviderSpec(path.Join(verifiedDir, ps.manifest))
		if err != nil {
			return fmt.Errorf("failed to get verified AzureProviderSpec: %w", err)
		}

		spec, err := m.GetAzureProviderSpec(path.Join(targetDir, ps.manifest))
		if err != nil {
			return fmt.Errorf("failed to get verified AzureProviderSpec: %w", err)
		}

		diff := missingElements(verifiedSpec.Permissions, spec.Permissions)
		if len(diff) == 0 {
			// If there are no new permission from verified version, we don't need to check roleDefinition.
			// This check is required because some credentials requests are using wildcards which is not allowed in the role definition.
			// We assume the verified version has all the required permissions, and if there's no update from the version,
			// we can also assume the target version has the required permissions.
			continue
		}

		rolePerms, err := m.GetRoleDefinitionPermission(ctx, ps.roleDefinitionId)
		if err != nil {
			return fmt.Errorf("failed to get role definition permission: %w", err)
		}
		if missingActions := missingElements(deref(rolePerms.Actions), spec.Permissions); len(missingActions) > 0 {
			fmt.Printf("\t%s: missing actions:\n", ps.name)
			for _, action := range missingActions {
				fmt.Printf("\t\t%s\n", action)
			}
			missing = true
		}

		if missingDataActions := missingElements(deref(rolePerms.DataActions), spec.DataPermissions); len(missingDataActions) > 0 {
			fmt.Printf("\t%s: missing data actions:\n", ps.name)
			for _, action := range missingDataActions {
				fmt.Printf("\t\t%s\n", action)
			}

			missing = true
		}
	}

	if missing {
		return &missingPermissionsError{}
	}
	return nil
}

func (m *manager) GetAzureProviderSpec(manifest string) (*cloudcredentialv1.AzureProviderSpec, error) {
	// Get the AzureProviderSpec from the credentials request
	f, err := os.Open(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}

	credreq, err := GetAzureCredentialsRequest(f)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure credentials request: %w", err)
	}

	var spec cloudcredentialv1.AzureProviderSpec
	err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &spec)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AzureProviderSpec: %w", err)
	}

	return &spec, nil
}

func (m *manager) GetRoleDefinitionPermission(ctx context.Context, roleDefId string) (*sdkauthorization.Permission, error) {
	// Get the role definition
	resp, err := m.roleDefinitions.Get(ctx, "", roleDefId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role definition: %w", err)
	}
	roleDef := resp.RoleDefinition

	// It's not guaranteed to have only one permission, but it's a good enough assumption for now.
	// Expect here to make sure the test fails if the assumption is wrong.
	if len(roleDef.Properties.Permissions) != 1 {
		return nil, fmt.Errorf("role definition has %d permissions, expected 1", len(roleDef.Properties.Permissions))
	}
	permission := roleDef.Properties.Permissions[0]

	return permission, nil
}

// missingElements enumerates elements in expected that are not in target
func missingElements(target, expected []string) []string {
	var result []string
	for _, x := range expected {
		if !slices.Contains(target, x) {
			result = append(result, x)
		}
	}
	return result
}

// deref converts a slice of pointers to a slice of strings
func deref(data []*string) []string {
	var result []string
	for _, d := range data {
		result = append(result, *d)
	}
	return result
}

func GetAzureCredentialsRequest(r io.Reader) (*cloudcredentialv1.CredentialsRequest, error) {
	dec := yaml.NewYAMLOrJSONDecoder(r, 4096)
	var credreq cloudcredentialv1.CredentialsRequest
	for {
		err := dec.Decode(&credreq)
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("azure credentials request not found")
		}
		if err != nil {
			return nil, err
		}
		unknown := runtime.Unknown{}
		err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &unknown)
		if err != nil {
			return nil, err
		}
		if unknown.Kind == "AzureProviderSpec" {
			return &credreq, nil
		}
	}
}
