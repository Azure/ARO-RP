package controller

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/operator/pkg/controller/genevalogging"
	"github.com/Azure/ARO-RP/operator/pkg/controller/internetchecker"
	"github.com/Azure/ARO-RP/operator/pkg/controller/pullsecret"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, pullsecret.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, internetchecker.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, genevalogging.Add)
}
