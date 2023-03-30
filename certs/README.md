# Certs

The AMEROOT cert is used in azure by the AppLens endpoint.  It needs to be added to the RPs Docker container so that GO Lang will not throw an error receiving a response from the endpoint in the RP.

## Verification

The AMEROOT cert was downloaded from [PKIs in azure](https://eng.ms/docs/products/onecert-certificates-key-vault-and-dsms/key-vault-dsms/autorotationandecr/cadetails).

The cert is currently good until May 24, 2026 and has a thumbprint of ```413E8AAC6049924B178BA636CBAF3963CCB963CD```.

The cert was verified by running ```openssl x509 -noout -fingerprint -sha1 -in AMEROOT_ameroot.crt```

The result of the oppenssl command was:
```sha1 Fingerprint=41:3E:8A:AC:60:49:92:4B:17:8B:A6:36:CB:AF:39:63:CC:B9:63:CD```
