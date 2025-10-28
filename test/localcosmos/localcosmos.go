package localcosmos

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/errorhandling"

	"github.com/containers/podman/v5/pkg/specgen"
)

const cosmosdbEmulatorPullspec = "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest"
const cosmosdbContainerName = "cosmosdb-emulator"

func GetPodmanConnection(ctx context.Context) (context.Context, error) {
	socket := os.Getenv("ARO_PODMAN_SOCKET")

	if socket == "" {
		sock_dir := os.Getenv("XDG_RUNTIME_DIR")
		socket = "unix:" + sock_dir + "/podman/podman.sock"
	}

	return bindings.NewConnection(ctx, socket)
}

func getContainerLogs(ctx context.Context, containerName string) error {
	stdout, stderr := make(chan string, 1024), make(chan string, 1024)
	go func() {
		for v := range stdout {
			fmt.Printf("stdout: %s", v)
		}
	}()

	go func() {
		for v := range stderr {
			fmt.Printf("stderr: %s", v)
		}
	}()
	err := containers.Logs(
		ctx,
		containerName,
		(&containers.LogOptions{}).WithStderr(true).WithStdout(true),
		stdout,
		stderr,
	)
	if err != nil {
		return fmt.Errorf("unable to get container logs: %w", err)
	}
	return nil
}

type LocalCosmosDB interface {
	Start(context.Context) error
	Stop() error
}

type localCosmos struct {
	conn context.Context
}

func NewLocalCosmos(conn context.Context) *localCosmos {
	return &localCosmos{
		conn: conn,
	}
}

func (lc *localCosmos) getContainer() (*define.InspectContainerData, error) {
	i, err := containers.Inspect(lc.conn, cosmosdbContainerName, nil)
	if err != nil {
		tgt := &errorhandling.ErrorModel{}
		if errors.As(err, &tgt) {
			if tgt.ResponseCode == http.StatusNotFound {
				return nil, nil
			}
		} else {
			return nil, fmt.Errorf("unable to inspect cosmosdb container: %w", err)
		}
	}
	return i, nil
}

func (lc *localCosmos) Start(ctx context.Context) error {
	options := (&images.PullOptions{}).
		WithQuiet(true).
		WithPolicy("missing")

	_, err := images.Pull(lc.conn, cosmosdbEmulatorPullspec, options)
	if err != nil {
		return fmt.Errorf("unable to pull cosmosdb image: %w", err)
	}

	i, err := lc.getContainer()
	if err != nil {
		return err
	}
	if i != nil {
		if i.State.Running {
			return nil
		}
		fmt.Printf("removing existing unhealthy cosmosdb container %s\n", i.ID)
		err = lc.stop(i.ID)
		if err != nil {
			return err
		}

	}

	s := specgen.NewSpecGenerator(cosmosdbEmulatorPullspec, false)
	s.Name = cosmosdbContainerName
	s.PortMappings = append(s.PortMappings,
		types.PortMapping{HostPort: 8081, ContainerPort: 8081},
		types.PortMapping{HostPort: 10250, ContainerPort: 10250, Range: 6},
	)

	s.Env = map[string]string{

		"AZURE_COSMOS_EMULATOR_ARGS":            "/enablepreview /DisableRateLimiting",
		"AZURE_COSMOS_EMULATOR_PARTITION_COUNT": "2"}

	fmt.Println("starting new cosmosdb container")

	container, err := containers.CreateWithSpec(lc.conn, s, nil)
	if err != nil {
		return fmt.Errorf("unable to create cosmosdb container: %w", err)
	}

	err = containers.Start(lc.conn, container.ID, nil)
	if err != nil {
		return fmt.Errorf("unable to start cosmosdb container: %w", err)
	}

	httpClient, databaseHostname := GetClient()

	nctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	for {
		time.Sleep(time.Second * 5)

		i, err := containers.Inspect(lc.conn, container.ID, nil)
		if err != nil {
			return fmt.Errorf("unable to inspect cosmosdb container: %w", err)
		}

		if i.State.Status == define.ContainerStateExited.String() || i.State.Status == define.ContainerStateStopped.String() {
			err2 := getContainerLogs(lc.conn, container.ID)
			if err2 != nil {
				fmt.Print(err2)
			}

			err3 := lc.stop(container.ID)
			if err3 != nil {
				fmt.Print(err3)
			}

			return fmt.Errorf("cosmosdb container exited unexpectedly: %v", container.Warnings)
		} else {
			req := &http.Request{
				Method: "HEAD",
				URL:    &url.URL{Scheme: "https", Host: databaseHostname, Path: "/_explorer/index.html"},
			}
			resp, err := httpClient.Do(req.WithContext(nctx))
			if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
				fmt.Printf("cosmosdb not ready yet: %v\n", err)
				continue
			} else {
				return nil
			}
		}
	}
}

func (lc *localCosmos) Stop() error {
	return lc.stop(cosmosdbContainerName)
}

func (lc *localCosmos) stop(containerID string) error {
	_, err := containers.Remove(lc.conn, containerID, &containers.RemoveOptions{Force: pointerutils.ToPtr(true)})
	if err != nil {
		return fmt.Errorf("unable to remove cosmosdb container: %w", err)
	}
	return nil
}

func GetClient() (*http.Client, string) {

	// Create HTTP client with custom transport
	httpClient := &http.Client{
		Transport: &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost: 20,
			// Skip TLS verification for local emulator with self-signed cert
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	// Read hostname from environment variable, this is specific to have local testing and CI testing
	host := os.Getenv("LOCAL_COSMOS_FOR_TEST_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	databaseHostname := host + ":8081"

	return httpClient, databaseHostname
}

func GetConnection(_env env.Core, m metrics.Emitter, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	logrusEntry := _env.LoggerForComponent("database")

	masterKey := "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==" // To-Do: move this outside the code
	dbAuthorizer, err := cosmosdb.NewMasterKeyAuthorizer(masterKey)
	if err != nil {
		return nil, err
	}

	handle, err := database.NewJSONHandle(aead)
	if err != nil {
		return nil, err
	}

	httpClient, databaseHostname := GetClient()

	return cosmosdb.NewDatabaseClient(logrusEntry, httpClient, handle, databaseHostname, dbAuthorizer), nil
}

func CleanupDB(ctx context.Context, client cosmosdb.DatabaseClient, dbName string) error {
	err := client.Delete(ctx, &cosmosdb.Database{ID: dbName})
	if err != nil {
		return fmt.Errorf("failure to cleanup test CosmosDB database: %w", err)
	}
	return nil
}

func CreateFreshDB(ctx context.Context, client cosmosdb.DatabaseClient) (string, error) {
	dbs, err := client.ListAll(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to list cosmosdb databases: %w", err)
	}

	for _, db := range dbs.Databases {
		err := client.Delete(ctx, db)
		if err != nil {
			return "", fmt.Errorf("failure to cleanup test CosmosDB database: %w", err)
		}
	}

	dbName := uuid.DefaultGenerator.Generate()

	db, err := client.Create(ctx, &cosmosdb.Database{ID: dbName})
	if err != nil {
		return "", fmt.Errorf("unable to create new CosmosDB database: %w", err)
	}

	collectionClient := cosmosdb.NewCollectionClient(client, dbName)

	_, err = collectionClient.Create(ctx, &cosmosdb.Collection{
		ID: "OpenShiftClusters",
		PartitionKey: &cosmosdb.PartitionKey{
			Paths: []string{"/partitionKey"},
			Kind:  cosmosdb.PartitionKeyKindHash,
		},
		UniqueKeyPolicy: &cosmosdb.UniqueKeyPolicy{
			UniqueKeys: []cosmosdb.UniqueKey{
				{
					Paths: []string{"/key"},
				},
				// These are in prod but are not really useful for local testing?
				// {
				// 	Paths: []string{"/clusterResourceGroupIdKey"},
				// },
				// {
				// 	Paths: []string{"/clientIdKey"},
				// },
			},
		},
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return "", fmt.Errorf("Failed to create OpenShiftClusters collection: %w", err)
	}

	triggerClient := cosmosdb.NewTriggerClient(collectionClient, "OpenShiftClusters")
	triggerClient.Create(ctx, &cosmosdb.Trigger{
		ID:               "renewLease",
		Body:             generator.RenewLeaseTriggerFunction,
		TriggerOperation: cosmosdb.TriggerOperationAll,
		TriggerType:      cosmosdb.TriggerTypePre,
	})

	_, err = collectionClient.Create(ctx, &cosmosdb.Collection{
		ID: "MaintenanceManifests",
		PartitionKey: &cosmosdb.PartitionKey{
			Paths: []string{"/clusterResourceID"},
			Kind:  cosmosdb.PartitionKeyKindHash,
		},
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return "", fmt.Errorf("Failed to create MaintenanceManifests collection: %w", err)
	}

	triggerClient = cosmosdb.NewTriggerClient(collectionClient, "MaintenanceManifests")
	triggerClient.Create(ctx, &cosmosdb.Trigger{
		ID:               "renewLease",
		Body:             generator.RenewLeaseTriggerFunction,
		TriggerOperation: cosmosdb.TriggerOperationAll,
		TriggerType:      cosmosdb.TriggerTypePre,
	})

	return db.ID, nil
}
