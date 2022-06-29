package ca

import (
	"crypto/x509"
	"time"

	"github.com/dapr/dapr/pkg/sentry/certs"
	"github.com/dapr/dapr/pkg/configmap"

)

// TrustRootBundle represents the root certificate, issuer certificate and their
// Respective expiry dates.
type TrustRootBundler interface {
	GetIssuerCertPem() []byte
	GetRootCertPem() []byte
	GetIssuerCertExpiry() *time.Time
	GetTrustAnchors() *x509.CertPool
	GetTrustDomain() string
}

type trustRootBundle struct {
	issuerCreds   *certs.Credentials
	trustAnchors  *x509.CertPool
	trustDomain   string
	rootCertPem   []byte
	issuerCertPem []byte
}

func (t *trustRootBundle) GetRootCertPem() []byte {
	return t.rootCertPem
}

func (t *trustRootBundle) GetIssuerCertPem() []byte {
	return t.issuerCertPem
}

func (t *trustRootBundle) GetIssuerCertExpiry() *time.Time {
	if t.issuerCreds == nil || t.issuerCreds.Certificate == nil {
		return nil
	}
	err := configmap.WriteToConfigMap("IssuerOrgName", t.issuerCreds.Certificate.Issuer.Organization[0])
	if err != nil {
		log.Fatalf("couldn't save issuer data to configmap, err: %s", err)
	}
	return &t.issuerCreds.Certificate.NotAfter
}

func (t *trustRootBundle) GetTrustAnchors() *x509.CertPool {
	return t.trustAnchors
}

func (t *trustRootBundle) GetTrustDomain() string {
	return t.trustDomain
}
