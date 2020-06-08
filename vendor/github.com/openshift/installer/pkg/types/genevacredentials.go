package types

// GenevaClusterLoggingCredentials contain the certificate and key needed to send logs from inside the cluster
type GenevaClusterLoggingCredentials struct {
	Certificate string
	Key         string
}
