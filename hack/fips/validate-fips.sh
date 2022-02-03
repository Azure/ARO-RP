#!/bin/bash

# The small go program below will validate that a 
# FIPS validated crypto lib
cat > ./hack/fips/main.go << 'EOF'
package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "crypto/tls/fipsonly"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func main() {
	log := utillog.GetLogger()
	log.Println("FIPS mode enabled")
}
EOF
trap "rm ./hack/fips/main.go" EXIT
echo "Attempting to run program that requires FIPS crypto"
go run ./hack/fips/main.go
