# Certificate rotation

First party certificate rotation for the following components is implemented in three different places:

- RP
- MDSD
- MDM

The first party certificate is stored in a keyvault. The certificate is
provided by Microsoft and in certain scenarios have to be rotated.

To ensure all three mentioned components read the new certificate,
following is implemented.


## RP

The certificate is read via [`certificateRefresher`](https://github.com/petrkotas/ARO-RP/blob/72b26b18ca43972770243809f09c33540c6ae8c9/pkg/env/certificateRefresher.go#L1), which regularly rereads the certificate from the keyvault and updates
the in-memory copy used in an authorizer.


## MDSD and MDM

Both MDSD and MDM, make use of regularly downloaded certificate. The certificate
is normally downloaded via [KeyVault extension](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/key-vault-linux).
Unfortunately in ARO RP VM uses RHEL which is unsupported Linux distribution.

Therefore a workaround is used. The [download systemd unit](https://github.com/Azure/ARO-RP/blob/4a48003b3e2345fda51ac3e860df4134cb494158/pkg/deploy/generator/resources_rp.go#L884) downloads the certificates and updates the correct file path

```
/var/lib/waagent/Microsoft.Azure.KeyVault.Store/
```

to mimic the KeyVault extension.

Moreover, both MDSD and MDM are deployed on VMs for the gateway and RP:

- `pkg/deploy/generator/resources_rp.go`
- `pkg/deploy/generator/resources_gateway.go`


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

The systemd unit `watch-mdm-credentials.path` monitors the file path for
changes and when the change occurs,
the MDM container is restarted forcing the re-read of the fresh certificate.