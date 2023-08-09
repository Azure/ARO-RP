package main

import "github.com/sirupsen/logrus"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func poc(log *logrus.Entry) error {
	log.Print("********** ARO-RP on AKS PoC **********")
	return nil
}
