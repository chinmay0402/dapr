package issuer

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/dapr/dapr/pkg/sentry/kubernetes"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/dapr/kit/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultSecretNamespace = "default"
)

var log = logger.NewLogger("dapr.sentry")

// Gets namespace for the ConfigMap
func getNamespace() string {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = defaultSecretNamespace
	}
	return namespace
}

// Fetches value of IssuerOrgName from ConfigMap to allow detection of dapr-generated certs
func GetIssuerMetadataFromConfigMap() string {
	log.Info("This function gets value from ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		log.Fatalf("could not get kubernetes client, err: %s", err)
	}
	namespace := getNamespace()
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "dapr-config-map", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to retrive configmap from kubernetes, err: %s", err)
	}
	issuerOrgName := configMap.Data["IssuerOrgName"]

	// TODO: look into some kind of retry mech
	
	return issuerOrgName
}

func RegisterActionToConfigMap(actionId string) error {
	log.Info("This function registers an action to ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		return err
	}
	namespace := getNamespace()
	configMapName := "dapr-config-map"
	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"actionId": actionId,
		},
	}

	if _, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{}); apiErrors.IsNotFound(err) { 
		// create configMap if not already present
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	} else {
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	}
	if err != nil {
		return errors.Wrap(err, "failed registering action id to kubernetes")
	}
	// _, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed saving issuer metadata to kubernetes")
	// }
	return nil
}

// check if some action is already present in key-value store
func CheckActionPresenceInConfigMap() string {
	log.Info("This function gets action id from ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		log.Fatalf("could not get kubernetes client, err: %s", err)
	}
	namespace := getNamespace()
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "dapr-config-map", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to retrieve configmap from kubernetes, err: %s", err)
	}
	actionId := configMap.Data["actionId"]

	// TODO: implement some kind of retry mech
	
	return actionId
}