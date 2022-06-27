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
	selfSignedRootCertLifetime = time.Hour * 56 // TODO: increase validity
	allowedClockSkew           = time.Minute * 2 // TODO: make 15 later
	daprGeneratedIssuerOrgName = "dapr.io/sentry"
)

var log = logger.NewLogger("dapr.monitor")

// Checks for presence of keywords in logs and takes appropriate actions
func ProcessLogs(logs string) {
	if strings.Contains(logs, "fatal") && (strings.Contains(logs, "x509") || strings.Contains(logs, "certificate") || strings.Contains(logs, "error from authenticator CreateSignedWorkloadCert")) {
	// if strings.Contains(logs, "x509") || strings.Contains(logs, "node-subscriber") {
		actionId := "1" // actionId for this scenario - other scenarios if added in the future should have unique action ids of their own

		if checkActionPresenceInConfigMap(actionId) != actionId { // check if actionId is present inside key-value store (TODO: change key-value store from ConfigMap to redis)
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
				log.Infof("generated new certificates")

				log.Infof("uploading certs to secrets...")
				err = certs.StoreRotatedCertsInKubernetes(rootCertPem, issuerCertPem, issuerKeyPem)
				if err != nil {
					log.Fatalf("could not upload certificate to k8s, err: %s", err)
				}
				log.Infof("certificate rotation successful, restarting pods...")

				// persist action taken to key-value store
				err = registerAction(actionId)
				if err != nil {
					log.Fatalf("couldn't register action to configmap, err: %s", err)
				}

				err = restartPods()
				if err != nil {
					log.Fatalf("couldn't restart pods, err: %s", err)
				}
				log.Infof("restart successful!")

				time.Sleep(2 * time.Minute) // TODO: sleep to make demo convenient, remove later

			} else {
				log.Fatalf("cannot auto rotate certs, issuer organization is: %s", issuerOrgName)
			}
		} else {
			log.Infof("action already taken")
		}
	}
}


// registers that an action of actionId was performed to the key-value store
func registerAction(actionId string) error {
	// currently register action to ConfigMap
	// TODO: replace ConfigMap by redis or some other state store having TTL support
	return issuer.RegisterActionToConfigMap(actionId)
}

// checks if an action was registered in a key-value store
func checkActionPresenceInConfigMap(actionId string) string {
	return issuer.CheckActionPresenceInConfigMap()
}
