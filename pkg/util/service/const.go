package service

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	DatabaseName        = "DATABASE_NAME"
	DatabaseAccountName = "DATABASE_ACCOUNT_NAME"
	KeyVaultPrefix      = "KEYVAULT_PREFIX"
	DBTokenUrl          = "DBTOKEN_URL"
)

type DB_TYPE int

const (
	_ DB_TYPE = iota
	DB_ALWAYS_MASTERKEY
	DB_ALWAYS_DBTOKEN
	DB_DBTOKEN_PROD_MASTERKEY_DEV
)
