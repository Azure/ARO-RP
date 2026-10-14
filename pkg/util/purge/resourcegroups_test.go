package purge

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"github.com/sirupsen/logrus"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestCleanResourceGroupsCleansAllGroupsBeforeDeleting(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.LogForTesting(t)
	ctrl := gomock.NewController(t)
	resourceGroups := mock_features.NewMockResourceGroupsClient(ctrl)
	securityGroups := mock_armnetwork.NewMockSecurityGroupsClient(ctrl)
	privateLinkServices := mock_armnetwork.NewMockPrivateLinkServicesClient(ctrl)

	groups := []mgmtfeatures.ResourceGroup{
		{Name: pointerutils.ToPtr("group1")},
		{Name: pointerutils.ToPtr("group2")},
	}
	var cleanedGroups atomic.Int32

	resourceGroups.EXPECT().List(ctx, "", nil).Return(groups, nil)
	for _, group := range groups {
		securityGroups.EXPECT().List(gomock.Any(), *group.Name, nil).Return([]*armnetwork.SecurityGroup(nil), nil)
		privateLinkServices.EXPECT().List(gomock.Any(), *group.Name, nil).DoAndReturn(func(context.Context, string, *armnetwork.PrivateLinkServicesClientListOptions) ([]*armnetwork.PrivateLinkService, error) {
			cleanedGroups.Add(1)
			return nil, nil
		})
	}
	resourceGroups.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, string) error {
		if cleanedGroups.Load() != int32(len(groups)) {
			t.Errorf("DeleteAndWait started before all resource groups were cleaned: got %d, want %d", cleanedGroups.Load(), len(groups))
		}
		return nil
	}).Times(len(groups))

	rc := &ResourceCleaner{
		log:                    log,
		resourcegroupscli:      resourceGroups,
		securitygroupscli:      securityGroups,
		privatelinkservicescli: privateLinkServices,
		shouldDelete:           func(mgmtfeatures.ResourceGroup, *logrus.Entry) bool { return true },
	}

	if err := rc.CleanResourceGroups(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRunResourceGroupTasksLimitsConcurrency(t *testing.T) {
	groups := make([]mgmtfeatures.ResourceGroup, maxConcurrentResourceGroupOperations+1)
	started := make(chan struct{}, len(groups))
	release := make(chan struct{})
	done := make(chan struct{})

	go func() {
		runResourceGroupTasks(groups, func(mgmtfeatures.ResourceGroup) error {
			started <- struct{}{}
			<-release
			return nil
		})
		close(done)
	}()

	for range maxConcurrentResourceGroupOperations {
		<-started
	}

	exceededLimit := false
	select {
	case <-started:
		exceededLimit = true
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	<-done

	if exceededLimit {
		t.Fatalf("more than %d resource group tasks ran concurrently", maxConcurrentResourceGroupOperations)
	}
}
