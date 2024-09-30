package oidchandler

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

// initTLSTransport initializes an HTTP client with a custom TLS configuration.
// It loads the CA certificate from the provided path and sets up the TLS transport.
// Returns an HTTP client or an error if the CA certificate cannot be read.
func initTLSTransport(caCertPath string) (*http.Client, error) {
	// load CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            caCertPool,
	}
	customTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	httpClient := &http.Client{
		Transport: customTransport,
	}
	return httpClient, nil
}
