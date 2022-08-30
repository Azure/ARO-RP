package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../util/mocks/pullsecretadmission
//go:generate go run ../../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../util/mocks/pullsecretadmission/requestdoer.go github.com/Azure/ARO-RP/pkg/operator/admission/validation/$GOPACKAGE RequestDoer
