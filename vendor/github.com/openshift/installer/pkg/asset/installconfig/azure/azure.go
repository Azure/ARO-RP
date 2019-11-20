package azure

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/installer/pkg/types/azure"

	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"

	azsub "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
)

const (
	defaultRegion string = "eastus"
)

// https://docs.microsoft.com/en-us/azure/architecture/best-practices/resource-naming#general
var resourceGroupNameRx = regexp.MustCompile(`(?i)^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

// Platform collects azure-specific configuration.
func Platform(credentials *Credentials) (*azure.Platform, error) {
	regions, err := getRegions(credentials)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of regions")
	}
	longRegions := make([]string, 0, len(regions))
	shortRegions := make([]string, 0, len(regions))
	for id, location := range regions {
		longRegions = append(longRegions, fmt.Sprintf("%s (%s)", id, location))
		shortRegions = append(shortRegions, id)
	}
	regionTransform := survey.TransformString(func(s string) string {
		return strings.SplitN(s, " ", 2)[0]
	})

	_, ok := regions[defaultRegion]
	if !ok {
		return nil, errors.Errorf("installer bug: invalid default azure region %q", defaultRegion)
	}

	sort.Strings(longRegions)
	sort.Strings(shortRegions)

	var region string
	err = survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Region",
				Help:    "The azure region to be used for installation.",
				Default: fmt.Sprintf("%s (%s)", defaultRegion, regions[defaultRegion]),
				Options: longRegions,
			},
			Validate: survey.ComposeValidators(survey.Required, func(ans interface{}) error {
				choice := regionTransform(ans).(string)
				i := sort.SearchStrings(shortRegions, choice)
				if i == len(shortRegions) || shortRegions[i] != choice {
					return errors.Errorf("invalid region %q", choice)
				}
				return nil
			}),
			Transform: regionTransform,
		},
	}, &region)
	if err != nil {
		return nil, err
	}

	var resourceGroupName string
	err = survey.Ask([]*survey.Question{
		{
			Prompt: &survey.Select{
				Message: "Resource group name",
				Help:    "The azure resource group to be used for installation.",
			},
			Validate: survey.ComposeValidators(survey.Required, func(ans interface{}) error {
				if !resourceGroupNameRx.MatchString(ans.(string)) {
					return errors.Errorf("invalid resource group %q", ans.(string))
				}
				return nil
			}),
		},
	}, &resourceGroupName)
	if err != nil {
		return nil, err
	}

	return &azure.Platform{
		Region:            region,
		ResourceGroupName: resourceGroupName,
	}, nil
}

func getRegions(credentials *Credentials) (map[string]string, error) {
	session, err := GetSession(credentials)
	if err != nil {
		return nil, err
	}
	client := azsub.NewClient()
	client.Authorizer = session.Authorizer
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	locations, err := client.ListLocations(ctx, session.Credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	locationsValue := *locations.Value
	allLocations := map[string]string{}
	for _, location := range locationsValue {
		allLocations[to.String(location.Name)] = to.String(location.DisplayName)
	}
	return allLocations, nil
}
