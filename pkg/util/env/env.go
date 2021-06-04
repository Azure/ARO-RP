package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "os"

// Wrapper around env calls so we can mock and test.
// https://gist.github.com/alexellis/adc67eb022b7fdca31afc0de6529e5ea

type OsEnv struct{}

type Environment struct{}

type EnvironmentSource interface {
	Getenv(key string) string
	LookupEnv(key string) (string, bool)
}

func (Environment) Getenv(source EnvironmentSource, key string) string {
	return source.Getenv(key)
}

func (Environment) LookupEnv(source EnvironmentSource, key string) (string, bool) {
	return source.LookupEnv(key)
}

func NewOsEnv() OsEnv {
	return OsEnv{}
}

func (OsEnv) Getenv(key string) string {
	return os.Getenv(key)
}
func (OsEnv) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
