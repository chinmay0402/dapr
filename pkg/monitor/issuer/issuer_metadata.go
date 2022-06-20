package issuer

import (
	"context"
	"os"

	"github.com/dapr/dapr/pkg/sentry/kubernetes"
	"github.com/dapr/kit/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultSecretNamespace = "default"
)

var log = logger.NewLogger("dapr.sentry")

func getNamespace() string {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = defaultSecretNamespace
	}
	return namespace
}

func GetIssuerMetadataFromConfigMap() string {
	log.Info("This function gets value from ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		log.Fatalf("could not get kkubernetes client, err: %s", err)
	}
	namespace := getNamespace()
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "dapr-config-map", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to retrive issuer metadata from kubernetes, err: %s", err)
	}
	issuerOrgName := configMap.Data["IssuerOrgName"]
	// TODO: implement some kind of retry mech
	return issuerOrgName
}