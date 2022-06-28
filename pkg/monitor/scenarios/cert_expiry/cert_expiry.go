package cert_expiry

import (
	"strings"
	"time"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/dapr/dapr/pkg/monitor/configmap"
	"github.com/dapr/dapr/pkg/sentry/ca"
	"github.com/dapr/dapr/pkg/sentry/certs"
	"github.com/dapr/kit/logger"
)

var log = logger.NewLogger("dapr.monitor")

const (
	selfSignedRootCertLifetime = time.Hour * 56 // TODO: increase validity
	allowedClockSkew           = time.Minute * 15 
	daprGeneratedIssuerOrgName = "dapr.io/sentry"
)

type CertExpiry struct{}

// Returns a new CertExpiry instance
func NewCertExpiry() CertExpiry {
	return CertExpiry{}
}

// Initiates Action for CertExpiry scenario
func (ce CertExpiry) Remediate() {
	actionId := "1" // actionId for this scenario - other scenarios if added in the future should have unique action ids of their own

	if ce.checkActionPresenceInConfigMap(actionId) != actionId { // check if actionId is present inside key-value store (TODO: change key-value store from ConfigMap to redis)
		log.Infof("Invalid certificate, renewal required")

		issuerOrgName := configmap.ReadKeyFromConfigMap("IssuerOrgName")

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
			err = ce.registerAction(actionId)
			if err != nil {
				log.Fatalf("couldn't register action to configmap, err: %s", err)
			}

			err = ce.restartPods()
			if err != nil {
				log.Fatalf("couldn't restart pods, err: %s", err)
			}
			log.Infof("restart successful!")

			time.Sleep(1 * time.Minute) 

		} else {
			log.Fatalf("cannot auto rotate certs, issuer organization is: %s", issuerOrgName)
		}
	} else {
		log.Infof("action already taken")
	}
}

// Detects the occurrence of CertExpiry scenario
func (ce CertExpiry) Detect(logs string) bool {
	if strings.Contains(logs, "fatal") {
		if strings.Contains(logs, "x509") ||  strings.Contains(logs, "certificate has expired") {
			return true
		}
		return true
	}
	return false
}

// Registers that an action of actionId was performed to the key-value store
func (ce CertExpiry) registerAction(actionId string) error {
	// currently register action to ConfigMap
	// TODO: replace ConfigMap by redis or some other state store having TTL support
	return configmap.WriteToConfigMap("actionId",actionId)
}

// Checks if an action was registered in a key-value store
func (ce CertExpiry) checkActionPresenceInConfigMap(actionId string) string {
	return configmap.ReadKeyFromConfigMap("actionId")
}

// Restarts sentry, operator and placement control plane services along with application pods
func (ce CertExpiry) restartPods() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})

	// restart sentry first
	data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().Deployments("dapr-system").Patch(context.Background(), "dapr-sentry", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		return err
	}
	log.Infof("sentry restart successful!")

	// restart operator deployment and placement server statefulset
	data = fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().Deployments("dapr-system").Patch(context.Background(), "dapr-operator", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		return err
	}
	log.Infof("operator restart successful!")

	data = fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().StatefulSets("dapr-system").Patch(context.Background(), "dapr-placement-server", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		return err
	}
	log.Infof("statefulset placement server restart successful!")
	
	// restart application pods
	for _, item := range deployments.Items {

		deploymentNamespace := item.GetObjectMeta().GetNamespace()
		deploymentName := item.GetObjectMeta().GetName()
		if deploymentNamespace != "default" { // take action only for default namespace - TODO: change later to handle non-dapr-system and non-default namespace applications as well
			continue
		}
		log.Infof("deployment namespace: %s, deployment name: %s", deploymentNamespace, deploymentName)

		data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
		_, err = clientset.AppsV1().Deployments(deploymentNamespace).Patch(context.Background(), deploymentName, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
		if err != nil {
			return err
		}
		log.Infof("restart successful!")
	}

	return nil
}