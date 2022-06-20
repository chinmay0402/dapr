package process

import (
	"strings"
	"time"

	"github.com/dapr/dapr/pkg/monitor/issuer"
	// "github.com/dapr/dapr/pkg/sentry/ca"
	// "github.com/dapr/dapr/pkg/sentry/certs"
	"github.com/dapr/kit/logger"
)

const (
	selfSignedRootCertLifetime = time.Hour * 8760
	allowedClockSkew 		   = time.Minute * 15
)
var log = logger.NewLogger("dapr.monitor")

func ProcessLogs(logs string) {
	// search for keywords?
	// how to create a logic flow?
	log.Infof("entered process logs")
	if strings.Contains(logs, "x509") || strings.Contains(logs, "error") || strings.Contains(logs, "Dashboard") {
		log.Infof("Invalid certificate, renewal required")
		// check if certs are dapr generated
		issuerOrgName := issuer.GetIssuerMetadataFromConfigMap()
		if issuerOrgName == "dapr.io/sentry" {
			log.Infof("auto rotating certs...")
		} else {
			log.Infof("cannot auto rotate certs, issuer organization is: %s", issuerOrgName)
		}
		// insert rotate certificate logic here
		// step 1 - create new certificates
			// substep 1 - generate private key
		/*
		rootKey, err := certs.GenerateECPrivateKey()
		if err != nil {
			log.Fatalf("could not generate new EC private key, err: %s", err)
		}
		_, rootCertPem, issuerCertPem, issuerKeyPem, err := ca.GetNewSelfSignedCertificates(
			rootKey, selfSignedRootCertLifetime, allowedClockSkew)
		if err != nil {
			log.Fatalf("could not get new self-signed certificates, err: %s", err)
		}
		log.Infof("generated new certificates, root.crt: %s\n issuer.crt: %s\n issuer.key: %s\n", rootCertPem, issuerCertPem, issuerKeyPem)
		// step 2 - store in k8s
		log.Infof("uploading certs to secrets...")
		err = certs.StoreRotatedCertsInKubernetes(rootCertPem, issuerCertPem, issuerKeyPem)
		if err != nil {
			log.Fatalf("could not upload certificate to k8s, err: %s", err)
		}
		// step 3 - log successful rotation
		log.Infof("certificate rotation successful!")
		*/
	}
}
