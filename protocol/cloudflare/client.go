package cloudflare

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/getlantern/keyman"
)

// CloudFlareClientProtocol implements clientProtocol using CloudFlare
type CloudFlareClientProtocol struct {
	upstreamHost         string
	cloudFlareHost       string
	upstreamAddr         string
	masqueradeCACertPool *x509.CertPool
}

func NewClientProtocol(upstreamHost string, upstreamPort int, masqueradeAs string, masqueradeCACert string) (*CloudFlareClientProtocol, error) {
	masqueradeCACertPool, err := poolForCert(masqueradeCACert)
	if err != nil {
		return nil, err
	}

	cloudFlareHost := upstreamHost
	if masqueradeAs != "" {
		cloudFlareHost = masqueradeAs
	}

	return &CloudFlareClientProtocol{
		upstreamHost:         upstreamHost,
		cloudFlareHost:       cloudFlareHost,
		upstreamAddr:         fmt.Sprintf("%s:%d", cloudFlareHost, upstreamPort),
		masqueradeCACertPool: masqueradeCACertPool,
	}, nil
}

func poolForCert(certString string) (*x509.CertPool, error) {
	if certString == "" {
		return nil, nil
	}
	log.Printf("Using masqueradeCACert: %s", certString)
	cert, err := keyman.LoadCertificateFromPEMBytes([]byte(certString))
	if err != nil {
		return nil, fmt.Errorf("Error loading masquerade CA cert from PEM bytes: %s", err)
	}
	return cert.PoolContainingCert(), nil
}

func (cf *CloudFlareClientProtocol) RewriteRequest(req *http.Request) {
	// Remember the host and scheme that was actually requested
	req.Header.Set(X_LANTERN_URL, req.URL.String())
	req.URL.Scheme = "http"

	// Set our upstream proxy as the host for this request
	req.Host = cf.upstreamHost
	req.URL.Host = cf.upstreamHost
}

func (cf *CloudFlareClientProtocol) RewriteResponse(resp *http.Response) {
}

func (cf *CloudFlareClientProtocol) Dial(addr string) (net.Conn, error) {
	tlsConfig := &tls.Config{
		RootCAs: cf.masqueradeCACertPool,
	}
	log.Printf("Using %s to handle request", cf.upstreamAddr)
	return tls.Dial("tcp", cf.upstreamAddr, tlsConfig)
}
