# Certificates and Secrets Explained

## Overview
This walks through all the keyvaults and explains the usage of the certificates and secrets used throughout.

## MDM/MDSD
Majority of the certificates below are mdm/mdsd related.  These certificates are certificates signed by the AME.GBL certificate authority and are vital to ensuring the necessary ingestion of metrics and logs within the ARO RP service and clusters.

More information about Geneva Monitoring can be found [here](https://eng.ms/docs/products/geneva/getting_started/newgettingstarted/overview).


## Certificates
Majority of the certificates are configured for auto-renewal to ensure that when nearing expiration, they are updated and rotated.  More information about certificate rotation can be found [here](./certificate-rotation.md)

## RP Keyvaults

1. Cluster (cls)
    - Certificates:
        - This keyvault contains all cluster `api` and `*.apps` certificates used within OpenShift.  These certificates are auto-rotated and pushed to clusters during AdminUpdates in the `configureAPIServerCertificate` and `configureIngressCertificate` steps.  These certificates will not be generated if the `DisableSignedCertificates` [feature flag](./feature-flags.md) is set within the RP config.

1. Portal (por)
    - Certificates:
        - `portal-client` is a certificate which is used within the aro-portal app registration.  The subject of this certificate must match that within the `trustedSubjects` section of the app registration manifest within the Azure portal, otherwise callbacks from the Microsoft AAD login service will not function correctly.
        - `portal-server` is a TLS certificate used in the SRE portal to access clusters
    - Secrets:
        - `portal-session-key` is a secret used to encrypt the session cookie when logging into the SRE portal.  When logging in, the SRE portal will encrypt a session cookie with this secret and push it to persist in your web browser.  Requests to the SRE portal then use this cookie to confirm authentication to the SRE portal.

1. Service (svc)
    - Certificates:
        - `cluster-mdsd` is the certificate persisted for logging for every ARO cluster
        - `rp-firstparty` is the certificate for the First Party service principal credentials
        - `rp-mdm` is the MDM certificate the RP uses to emit cluster metrics within the monitor and RP metrics within the RP processes
        - `rp-mdsd` is the MDSD certificate the RP uses to emit logs to the Geneva/MDSD service
        - `rp-server` is the TLS certificate used for RP RESTful HTTPS calls
    - Secrets:
        - `encryption-key` a legacy secret which uses the old encryption suites to encrypt secure strings and secure bytes within the cluster document
        - `encryption-key-v2` the new secret used to encrypt secure strings and secure bytes within the cluster document
        - `fe-encryption-key` a legacy secret used to encrypt `skipTokens` for paging OpenShiftCluster List requests.  Uses an older encryption suite.
        - `fe-encryption-key-v2` a new secret used to encrypt `skipTokens` for paging OpenShiftCluster List requests

## Gateway Keyvaults

1. Gateway (gwy)
    - Certificates:
        - `gwy-mdm` the certificate used for emitting metrics to the Geneva/MDM service
        - `gwy-mdsd` the certificate used for emitting logs to the Geneva/MDSD service

