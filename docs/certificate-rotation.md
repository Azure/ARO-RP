# Certificate rotation

The first party certificate and certificates for telemetry services are stored
in Azure KeyVault. The certificates are provided by Microsoft and in certain
scenarios have to be rotated.


## RP

The certificate is read via [`certificateRefresher`](https://github.com/petrkotas/ARO-RP/blob/72b26b18ca43972770243809f09c33540c6ae8c9/pkg/env/certificateRefresher.go#L1), which regularly rereads the certificate from the keyvault and updates
the in-memory copy used in an authorizer.

## Telemetry

The certificates are refreshed on-disk via [a systemd timer](https://github.com/Azure/ARO-RP/blob/da67a1057257d8f8a6b2945b81730ca89b23892e/pkg/deploy/generator/scripts/util-services.sh#L1208).

### MDSD

MDSD uses the configuration to read new keys automatically. It read from the
known file path

```
/var/lib/waagent/Microsoft.Azure.KeyVault.Store/
```

to get the fresh certificate.


### MDM

MDM currently does not have the ability to read fresh certificate.
The certificate is read from known path, but it is not re-read.
To overcome this limitation, new systemd unit is introduced.

The systemd unit
[`watch-mdm-credentials.path`](https://github.com/Azure/ARO-RP/blob/da67a1057257d8f8a6b2945b81730ca89b23892e/pkg/deploy/generator/scripts/util-services.sh#L1403)
monitors the file path for changes and when the change occurs, the MDM container
is restarted forcing the re-read of the fresh certificate.

### OTel (Gateway)

The TLS certificate used for serving the OTel-GRPC service is monitored by
[`watch-gateway-otel-credentials.path`](https://github.com/Azure/ARO-RP/blob/da67a1057257d8f8a6b2945b81730ca89b23892e/pkg/deploy/generator/scripts/util-services.sh#L1433)
and restarts the OTel service, similar to MDM.
