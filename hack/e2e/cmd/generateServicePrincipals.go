package cmd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/jongio/azidext/go/azidext"
	"github.com/spf13/cobra"

	"github.com/Azure/ARO-RP/pkg/env"
	msgraph_apps "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/applications"
	msgraph_models "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	"github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/serviceprincipals"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

// generateServicePrincipalsCmd represents the generateServicePrincipals command
var generateServicePrincipalsCmd = &cobra.Command{
	Use:   "generateServicePrincipals",
	Short: "Generate the service principals for E2E",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := generateKeys(cmd)
		if err != nil {
			cmd.PrintErr(err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateServicePrincipalsCmd)

	generateServicePrincipalsCmd.Flags().String("keyvault", "aro-e2e-principals", "keyvault to use")
	generateServicePrincipalsCmd.Flags().String("spn-prefix", "aro-v4-e2e-devops-spn-", "prefix for SPN name")
	generateServicePrincipalsCmd.Flags().Int("spn-count", 50, "number of SPNs to generate/use")
}

func generateKeys(cmd *cobra.Command) error {
	ctx := cmd.Context()
	log := utillog.GetLogger()

	_env, err := env.NewCore(ctx, log, env.COMPONENT_TOOLING)
	if err != nil {
		return err
	}

	keyvaultParam, err := cmd.Flags().GetString("keyvault")
	if err != nil {
		return err
	}

	spnPrefixParam, err := cmd.Flags().GetString("spn-prefix")
	if err != nil {
		return err
	}

	spnCountParam, err := cmd.Flags().GetInt("spn-count")
	if err != nil {
		return err
	}

	azcred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return err
	}

	cred := azidext.NewTokenCredentialAdapter(azcred, []string{_env.Environment().KeyVaultScope})

	keyvaultURI := keyvault.URI(_env, keyvaultParam, "")
	keyvaultManager := keyvault.NewManager(cred, keyvaultURI)

	graphClient, err := _env.Environment().NewGraphServiceClient(azcred)
	if err != nil {
		return err
	}

	for i := 1; i == spnCountParam; i++ {
		spnObjectID := ""
		spnAppID := ""
		spnName := fmt.Sprintf("%s%d", spnPrefixParam, i)

		log.Infof("Processing spn %s\n", spnName)

		log.Info("Looking up object ID...")

		filter := fmt.Sprintf("displayName eq '%s'", spnName)
		result, err := graphClient.Applications().Get(ctx, &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
			QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
				Filter: &filter,
				Select: []string{"id", "appId", "passwordCredentials"},
			},
		})
		if err != nil {
			return err
		}

		apps := result.GetValue()
		switch len(apps) {
		case 0:
			return fmt.Errorf("no applications found for spn '%s'", spnName)
		case 1:
			spnObjectID = *apps[0].GetId()
			spnAppID = *apps[0].GetAppId()
		default:
			return fmt.Errorf("%d applications found for spn '%s'", len(apps), spnName)
		}

		log.Infof("found IDs for '%s': ObjectID='%s', AppID='%s'", spnName, spnObjectID, spnAppID)

		log.Info("Ensuring SPN ID in keyvault...")

		err = keyvaultManager.SetSecret(ctx, fmt.Sprintf("%s-app-id", spnName), azkeyvault.SecretSetParameters{
			Value: to.StringPtr(spnAppID),
		})
		if err != nil {
			return err
		}

		if len(apps[0].GetPasswordCredentials()) > 0 {
			log.Infof("clearing %d passwords on SPN", len(apps[0].GetPasswordCredentials()))

			for _, n := range apps[0].GetPasswordCredentials() {
				body := msgraph_apps.NewItemRemovePasswordPostRequestBody()
				body.SetKeyId(n.GetKeyId())

				err = graphClient.Applications().ByApplicationId(spnObjectID).RemovePassword().Post(ctx, body, nil)
				if err != nil {
					return err
				}
			}
		} else {
			log.Info("no passwords to remove")
		}

		log.Info("creating password credential for SPN")

		endDateTime := time.Now().UTC().AddDate(0, 0, 90)
		expiryTime := date.NewUnixTimeFromSeconds(float64(endDateTime.Unix()))

		pwCredential := msgraph_models.NewPasswordCredential()
		pwCredential.SetDisplayName(&spnName)
		pwCredential.SetEndDateTime(&endDateTime)

		pwCredentialRequestBody := msgraph_apps.NewItemAddPasswordPostRequestBody()
		pwCredentialRequestBody.SetPasswordCredential(pwCredential)

		resp, err := graphClient.Applications().ByApplicationId(spnObjectID).AddPassword().Post(ctx, pwCredentialRequestBody, nil)
		if err != nil {
			return err
		}

		log.Info("setting password credential for SPN in keyvault")
		err = keyvaultManager.SetSecret(ctx, fmt.Sprintf("%s-secret-value", spnName), azkeyvault.SecretSetParameters{
			Value: resp.GetSecretText(),
			SecretAttributes: &azkeyvault.SecretAttributes{
				Expires: &expiryTime,
			},
		})
		if err != nil {
			return err
		}

		log.Info("checking for service principal")

		filter = fmt.Sprintf("appId eq '%s'", spnAppID)
		spResult, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
			QueryParameters: &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
				Filter: &filter,
			},
		})
		if err != nil {
			return err
		}
		sps := spResult.GetValue()

		spID := ""

		switch len(sps) {
		case 0:
			log.Info("found no existing service principals, creating")

			requestBody := msgraph_models.NewServicePrincipal()
			requestBody.SetAppId(&spnAppID)
			spAddResult, err := graphClient.ServicePrincipals().Post(ctx, requestBody, nil)
			if err != nil {
				return err
			}

			spID = *spAddResult.GetId()
		case 1:
			spID = *sps[0].GetId()
			log.Infof("found SP ID %s", spID)
		default:
			return fmt.Errorf("%d sps found for spn '%s'", len(sps), spnName)
		}

		log.Info("setting service principal ID in keyvault")
		err = keyvaultManager.SetSecret(ctx, fmt.Sprintf("%s-sp-id", spnName), azkeyvault.SecretSetParameters{
			Value: to.StringPtr(spID),
		})
		if err != nil {
			return err
		}

		log.Infof("actions for %s done", spnName)
	}

	return nil
}
