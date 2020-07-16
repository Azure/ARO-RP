package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Generator defines generator main interface
type Generator interface {
	RPTemplate() (map[string]interface{}, error)
	RPGlobalTemplate() (map[string]interface{}, error)
	RPGlobalSubscriptionTemplate() (map[string]interface{}, error)
	RPSubscriptionTemplate() (map[string]interface{}, error)
	ManagedIdentityTemplate() (map[string]interface{}, error)
	PreDeployTemplate() (map[string]interface{}, error)
}

type generator struct {
	production bool
}

func New(production bool) Generator {
	return &generator{
		production: production,
	}
}
