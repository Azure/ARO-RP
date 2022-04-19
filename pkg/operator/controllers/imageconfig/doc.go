package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

/*

The controller in this package aims to ensure that ImageConfig manifest
allows / does not block images registries that are essential to smooth cluster operation.

There is one flag which controls the operations performed by this controller:

aro.imageconfig.enabled:
- When set to false, the controller will noop and not perform any further action
- When set to true, the controller will attempt to reconcile the image.config manifest

If aro.imageconfig.enabled=true and manifest contains AllowedRegistries:
- The controller will ensure the required registries are part of allowed registries, so as to allow traffic from essential registries

If aro.imageconfig.enabled=true and manifest contains Blocked Registries:
- The controller will ensure the required registries are NOT part of blocked registries

If aro.imageconfig.enabled=true and manifest contains both AllowedRegistries & BlockedRegistries:
- The controller will fail silently and not requeue as this is not a supported action

More information on image.config resource can be found here:
https://docs.openshift.com/container-platform/4.6/openshift_images/image-configuration.html

*/
