package cluster

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func RotateACRToken(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	force := false

	tokensClient, err := th.TokensClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	registriesClient, err := th.RegistriesClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	manager, err := acrtoken.NewManager(th.Environment(), tokensClient, registriesClient)
	if err != nil {
		return mimo.TerminalError(err)
	}

	return rotateACRTokenWithManager(th, manager, force)
}

func rotateACRTokenWithManager(th mimo.TaskContext, manager acrtoken.Manager, force bool) error {
	ch, err := th.ClientHelper()
	if err != nil {
		return mimo.TerminalError(err)
	}
	err = cluster.RotateACRToken(th, th.Environment(), th.Log(), ch, th.GetOpenShiftClusterDocument(), manager, th.PatchOpenShiftClusterDocument, force)
	if err != nil {
		return mimo.TransientError(err)
	}

	return nil
}
