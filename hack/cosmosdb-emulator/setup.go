package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// This tool reads the ARM template from pkg/deploy/assets/databases-development.json
// and creates the corresponding database structure in the local Cosmos DB emulator.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

type ARMTemplate struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	APIVersion string                 `json:"apiVersion"`
	DependsOn  []string               `json:"dependsOn,omitempty"`
	Properties map[string]interface{} `json:"properties"`
}

func main() {
	ctx := context.Background()
	log := logrus.New().WithField("component", "cosmosdb-setup")

	templatePath := "pkg/deploy/assets/databases-development.json"
	template, err := readARMTemplate(templatePath)
	if err != nil {
		log.Fatalf("Failed to read ARM template: %v", err)
	}

	dbClient, err := database.NewLocalDatabaseClient(log, &noop.Noop{}, nil)
	if err != nil {
		log.Fatalf("Failed to create database client: %v", err)
	}

	dbName := os.Getenv("COSMOSDB_EMULATOR_DATABASE_NAME")
	if dbName == "" {
		dbName = "ARO"
	}

	var hasErrors bool

	for _, resource := range template.Resources {
		if resource.Type == "Microsoft.DocumentDB/databaseAccounts/sqlDatabases" {
			log.Infof("Creating database: %s", dbName)
			if err := createDatabase(ctx, dbClient, dbName); err != nil {
				log.Errorf("Failed to create database: %v", err)
				hasErrors = true
			}
			time.Sleep(2 * time.Second)
			break
		}
	}

	createdCollections := make(map[string]bool)
	for _, resource := range template.Resources {
		if resource.Type == "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers" {
			if err := createCollection(ctx, dbClient, dbName, resource); err != nil {
				log.Errorf("Error creating collection: %v", err)
				hasErrors = true
			} else {
				// Track successfully created collections
				if resData, ok := resource.Properties["resource"].(map[string]interface{}); ok {
					if id, ok := resData["id"].(string); ok {
						createdCollections[id] = true
					}
				}
			}
			time.Sleep(2 * time.Second)
		}
	}

	for _, resource := range template.Resources {
		if resource.Type == "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/triggers" {
			collectionName := extractCollectionFromDependencies(resource.DependsOn)
			if collectionName == "" || !createdCollections[collectionName] {
				log.Warnf("Skipping trigger creation - collection %s does not exist", collectionName)
				continue
			}

			if err := createTrigger(ctx, dbClient, dbName, resource); err != nil {
				log.Errorf("Error creating trigger: %v", err)
				hasErrors = true
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	if hasErrors {
		log.Fatal("Database setup completed with errors")
	} else {
		log.Info("Database setup completed successfully!")
	}
}

func readARMTemplate(path string) (*ARMTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var template ARMTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

func createDatabase(ctx context.Context, client cosmosdb.DatabaseClient, dbName string) error {
	db := &cosmosdb.Database{
		ID: dbName,
	}

	_, err := client.Create(ctx, db)
	if err != nil && !strings.Contains(err.Error(), "Conflict") {
		return err
	}

	return nil
}

func createCollection(ctx context.Context, client cosmosdb.DatabaseClient, dbName string, resource Resource) error {
	resourceData, ok := resource.Properties["resource"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid resource properties for collection")
	}

	collectionID, ok := resourceData["id"].(string)
	if !ok {
		return fmt.Errorf("collection id not found in resource properties")
	}

	collection := &cosmosdb.Collection{
		ID: collectionID,
	}

	if partitionKey, ok := resourceData["partitionKey"].(map[string]interface{}); ok {
		if paths, ok := partitionKey["paths"].([]interface{}); ok && len(paths) > 0 {
			collection.PartitionKey = &cosmosdb.PartitionKey{
				Paths: []string{paths[0].(string)},
				Kind:  "Hash",
			}
		}
	}

	if uniqueKeyPolicy, ok := resourceData["uniqueKeyPolicy"].(map[string]interface{}); ok {
		if uniqueKeys, ok := uniqueKeyPolicy["uniqueKeys"].([]interface{}); ok {
			collection.UniqueKeyPolicy = &cosmosdb.UniqueKeyPolicy{
				UniqueKeys: make([]cosmosdb.UniqueKey, 0, len(uniqueKeys)),
			}

			for _, uk := range uniqueKeys {
				if ukMap, ok := uk.(map[string]interface{}); ok {
					if paths, ok := ukMap["paths"].([]interface{}); ok {
						uniqueKey := cosmosdb.UniqueKey{
							Paths: make([]string, 0, len(paths)),
						}
						for _, p := range paths {
							uniqueKey.Paths = append(uniqueKey.Paths, p.(string))
						}
						collection.UniqueKeyPolicy.UniqueKeys = append(collection.UniqueKeyPolicy.UniqueKeys, uniqueKey)
					}
				}
			}
		}
	}

	if ttl, ok := resourceData["defaultTtl"]; ok {
		logrus.Warnf("Collection %s specifies defaultTtl=%v but this is not supported by the SDK", collectionID, ttl)
	}

	logrus.Infof("Creating collection: %s", collectionID)
	collectionClient := cosmosdb.NewCollectionClient(client, dbName)
	_, err := collectionClient.Create(ctx, collection)
	if err != nil && !strings.Contains(err.Error(), "Conflict") {
		return fmt.Errorf("failed to create collection %s: %v", collectionID, err)
	}

	return nil
}

func createTrigger(ctx context.Context, client cosmosdb.DatabaseClient, dbName string, resource Resource) error {
	resourceData, ok := resource.Properties["resource"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid resource properties for trigger")
	}

	triggerID, ok := resourceData["id"].(string)
	if !ok {
		return fmt.Errorf("trigger id not found in resource properties")
	}

	collectionName := extractCollectionFromDependencies(resource.DependsOn)
	if collectionName == "" {
		return fmt.Errorf("could not determine collection for trigger %s", triggerID)
	}

	trigger := &cosmosdb.Trigger{
		ID:               triggerID,
		Body:             resourceData["body"].(string),
		TriggerType:      cosmosdb.TriggerType(resourceData["triggerType"].(string)),
		TriggerOperation: cosmosdb.TriggerOperation(resourceData["triggerOperation"].(string)),
	}

	logrus.Infof("Creating trigger: %s on collection %s", triggerID, collectionName)

	collectionClient := cosmosdb.NewCollectionClient(client, dbName)
	triggerClient := cosmosdb.NewTriggerClient(collectionClient, collectionName)

	_, err := triggerClient.Create(ctx, trigger)
	if err != nil && !strings.Contains(err.Error(), "Conflict") {
		return fmt.Errorf("failed to create trigger %s on %s: %v", triggerID, collectionName, err)
	}

	return nil
}

func extractCollectionFromDependencies(dependencies []string) string {
	for _, dep := range dependencies {
		if strings.Contains(dep, "/containers',") && strings.Contains(dep, "resourceId(") {
			parts := strings.Split(dep, ",")
			if len(parts) >= 4 {
				collectionPart := strings.TrimSpace(parts[len(parts)-1])
				collectionName := strings.Trim(collectionPart, " '\")]")
				return collectionName
			}
		}
	}
	return ""
}
