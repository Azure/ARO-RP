# Encryption

## Algorithm

ARO RP encryption is using ChaCha20-Poly1305 AEAD.
The rationale behind it:

1. AE is intended to safely provide authentication and encryption as a single mode with a single key
2. GCM is included natively in Golang but the standard has a short-ish nonce; either a counter needs to be persisted or NIST recommends not using more than 2^32 random nonces per key.
3. ChaCha20-Poly1305 is performant and well used (e.g. in TLS), it has a longer nonce which avoids the problem above.  It is implemented in the Golang supplementary crypto library.

Note that the entire text must be in memory for encrypting/decrypt operations - should be OK given that the graph is not huge (typically around 1.7MB, but also pretty compressible if wanted).

Implementation for it can be found in `pkg/encrypt`

## Code implementation

Database encryption for ARO RP is done in 2 location, depending on the fields
and purpose.

1. `pkg/encrypt` - encrypts individual fields of the document. This function is
called in database client on read/writes operations to do this before database
writes.
2. `pkg/install` is using `encrypt` package and encrypts graph, stored in storage
blob.

It uses 3 methods:

1. Custom Marshaler for `rsa.PrivateKey`
2. Individual fields marshaling
3. Graph blob encrypt

## Encrypted fields

Transparently encrypt the following fields in the database:
```
/openShiftCluster/properties/servicePrincipalProfile/clientSecret
/openShiftCluster/properties/sshKey
/openShiftCluster/properties/adminKubeconfig
/openShiftCluster/properties/kubeadminPassword
```

Transparently encrypt the cluster graph (cluster storage account, aro container, graph blob)
