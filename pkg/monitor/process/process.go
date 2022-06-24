package process

import (
	"strings"
	"time"

	"github.com/dapr/dapr/pkg/monitor/issuer"
	"github.com/dapr/dapr/pkg/sentry/ca"
	"github.com/dapr/dapr/pkg/sentry/certs"
	"github.com/dapr/kit/logger"
)

const (
	selfSignedRootCertLifetime = time.Hour * 56
	allowedClockSkew           = time.Minute * 15
	daprGeneratedIssuerOrgName = "dapr.io/sentry"
)

var log = logger.NewLogger("dapr.monitor")

// Checks for presence of keywords in logs and takes appropriate actions
func ProcessLogs(logs string) {
	// if strings.Contains(logs, "fatal") && (strings.Contains(logs, "x509") || strings.Contains(logs, "error from authenticator CreateSignedWorkloadCert")) {
	if strings.Contains(logs, "x509") || strings.Contains(logs, "node-subscriber") {
		actionId := "1"
		if getAction(actionId) != actionId {
			log.Infof("Invalid certificate, renewal required")

			issuerOrgName := issuer.GetIssuerMetadataFromConfigMap()

			// check if certs are dapr generated
			if issuerOrgName == daprGeneratedIssuerOrgName {
				log.Infof("auto rotating certs...")
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

				log.Infof("uploading certs to secrets...")
				err = certs.StoreRotatedCertsInKubernetes(rootCertPem, issuerCertPem, issuerKeyPem)
				if err != nil {
					log.Fatalf("could not upload certificate to k8s, err: %s", err)
				}

				log.Infof("certificate rotation successful, restart required...")

				// res, err := kubeClient.CoreV1().Secrets(namespace).Get()

				// persist action taken
				registerAction(actionId)

				// TODO: insert restart logic
				restart()
				time.Sleep(600 * time.Second)

			} else {
				log.Fatalf("cannot auto rotate certs, issuer organization is: %s", issuerOrgName)
			}
		} else {
			log.Infof("action already taken")
		}
	}
}

func registerAction(actionId string) {
	// currently register action to ConfigMap
	// TODO: replace ConfigMap by redis or some other state store having TTL support
	issuer.RegisterActionToConfigMap(actionId)
}

func getAction(actionId string) string {
	return issuer.CheckActionPresenceInConfigMap()
}
