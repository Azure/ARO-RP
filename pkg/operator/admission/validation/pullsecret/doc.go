package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//This package implements a validation webhook for pullsecrets.
//You can find documentation on validating webhooks here
//https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
//Validating webhooks can either accept or reject an object
//and only the final object (after all mutating webhooks are
//called).

//This will check that all the required credentials present in the
//pullsecret are providing a token when used against the registries'
//URLs.

//It is deployed by the operator in cmd/aro/operator.go using the
//standard deployer. It uses openshift service certificates
//as it is required for webhooks to use https.
//You can find more info about service certificates at
//https://docs.openshift.com/container-platform/4.10/security/certificates/service-serving-certificate.html
