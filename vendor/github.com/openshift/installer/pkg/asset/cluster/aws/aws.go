// Package aws extracts AWS metadata from install configurations.
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/types"
	awstypes "github.com/openshift/installer/pkg/types/aws"
)

// Metadata converts an install configuration to AWS metadata.
func Metadata(clusterID, infraID string, config *types.InstallConfig) *awstypes.Metadata {
	return &awstypes.Metadata{
		Region: config.Platform.AWS.Region,
		Identifier: []map[string]string{{
			fmt.Sprintf("kubernetes.io/cluster/%s", infraID): "owned",
		}, {
			"openshiftClusterID": clusterID,
		}},
		ServiceEndpoints: config.AWS.ServiceEndpoints,
		ClusterDomain:    config.ClusterDomain(),
	}
}

// PreTerraform performs any infrastructure initialization which must
// happen before Terraform creates the remaining infrastructure.
func PreTerraform(ctx context.Context, clusterID string, installConfig *installconfig.InstallConfig) error {

	if err := tagSharedVPCResources(ctx, clusterID, installConfig); err != nil {
		return err
	}

	if err := tagSharedIAMRoles(ctx, clusterID, installConfig); err != nil {
		return err
	}

	return nil
}

func tagSharedVPCResources(ctx context.Context, clusterID string, installConfig *installconfig.InstallConfig) error {
	if len(installConfig.Config.Platform.AWS.Subnets) == 0 {
		return nil
	}

	privateSubnets, err := installConfig.AWS.PrivateSubnets(ctx)
	if err != nil {
		return err
	}

	publicSubnets, err := installConfig.AWS.PublicSubnets(ctx)
	if err != nil {
		return err
	}

	ids := make([]*string, 0, len(privateSubnets)+len(publicSubnets))
	for id := range privateSubnets {
		ids = append(ids, aws.String(id))
	}
	for id := range publicSubnets {
		ids = append(ids, aws.String(id))
	}

	session, err := installConfig.AWS.Session(ctx)
	if err != nil {
		return errors.Wrap(err, "could not create AWS session")
	}

	tagKey, tagValue := sharedTag(clusterID)

	ec2Client := ec2.New(session, aws.NewConfig().WithRegion(installConfig.Config.Platform.AWS.Region))
	if _, err = ec2Client.CreateTagsWithContext(ctx, &ec2.CreateTagsInput{
		Resources: ids,
		Tags:      []*ec2.Tag{{Key: &tagKey, Value: &tagValue}},
	}); err != nil {
		return errors.Wrap(err, "could not add tags to subnets")
	}

	if zone := installConfig.Config.AWS.HostedZone; zone != "" {
		route53Client := route53.New(session)
		if _, err := route53Client.ChangeTagsForResourceWithContext(ctx, &route53.ChangeTagsForResourceInput{
			ResourceType: aws.String("hostedzone"),
			ResourceId:   aws.String(zone),
			AddTags:      []*route53.Tag{{Key: &tagKey, Value: &tagValue}},
		}); err != nil {
			return errors.Wrap(err, "could not add tags to hosted zone")
		}
	}

	return nil
}

func tagSharedIAMRoles(ctx context.Context, clusterID string, installConfig *installconfig.InstallConfig) error {
	var iamRoleNames []*string
	if mp := installConfig.Config.ControlPlane; mp != nil {
		awsMP := &awstypes.MachinePool{}
		awsMP.Set(installConfig.Config.AWS.DefaultMachinePlatform)
		awsMP.Set(mp.Platform.AWS)
		if iamRole := awsMP.IAMRole; iamRole != "" {
			iamRoleNames = append(iamRoleNames, &iamRole)
		}
	}
	if mp := installConfig.Config.WorkerMachinePool(); mp != nil {
		awsMP := &awstypes.MachinePool{}
		awsMP.Set(installConfig.Config.AWS.DefaultMachinePlatform)
		awsMP.Set(mp.Platform.AWS)
		if iamRole := awsMP.IAMRole; iamRole != "" {
			iamRoleNames = append(iamRoleNames, &iamRole)
		}
	}

	if len(iamRoleNames) == 0 {
		return nil
	}

	session, err := installConfig.AWS.Session(ctx)
	if err != nil {
		return err
	}
	key, value := sharedTag(clusterID)
	iamTags := []*iam.Tag{{Key: &key, Value: &value}}
	iamClient := iam.New(session, aws.NewConfig().WithRegion(installConfig.Config.Platform.AWS.Region))

	for _, iamRoleName := range iamRoleNames {
		if _, err := iamClient.TagRoleWithContext(ctx, &iam.TagRoleInput{
			RoleName: iamRoleName,
			Tags:     iamTags,
		}); err != nil {
			return errors.Wrapf(err, "could not tag %s role", *iamRoleName)
		}
	}

	return nil
}

func sharedTag(clusterID string) (string, string) {
	return fmt.Sprintf("kubernetes.io/cluster/%s", clusterID), "shared"
}
