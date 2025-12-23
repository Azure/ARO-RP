package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	envOpenShiftVersions                = "OPENSHIFT_VERSIONS"
	envInstallerImageDigests            = "INSTALLER_IMAGE_DIGESTS"
	envPlatformWorkloadIdentityRoleSets = "PLATFORM_WORKLOAD_IDENTITY_ROLE_SETS"

	// pprof configuration environment variables
	// PPROF_ENABLED: Set to "true" or "1" to enable pprof server.
	//                Defaults to enabled only in development mode (RP_MODE=development).
	envPprofEnabled = "PPROF_ENABLED"

	// PPROF_PORT: TCP port for the pprof HTTP server.
	//             Defaults to 6060.
	envPprofPort = "PPROF_PORT"

	// PPROF_HOST: Host address for the pprof server.
	//             Restricted to localhost addresses for security.
	//             Defaults to 127.0.0.1.
	envPprofHost = "PPROF_HOST"
)
