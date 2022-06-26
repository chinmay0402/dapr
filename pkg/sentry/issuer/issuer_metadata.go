package issuer

import (
	"context"
	"os"

	"github.com/dapr/dapr/pkg/sentry/kubernetes"
	"github.com/dapr/kit/logger"
	"github.com/pkg/errors"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultSecretNamespace = "default"
	configMapName 		   = "dapr-config-map"
)

var log = logger.NewLogger("dapr.sentry")

// Gets namespace in which the ConfigMap is present
func getNamespace() string {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = defaultSecretNamespace
	}
	return "dapr-system"
}

// Writes the issuerOrgName to ConfigMap so that it can be fetched by dapr-monitor
func WriteIssuerMetadataToConfigMap(issuerOrgName string) error {
	log.Info("This function writes a value to ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		return err
	}
	namespace := getNamespace()
	configMapName := "dapr-config-map"
	currentConfigMap := getConfigMap() // get config map
	log.Infof("got config map")
	currentConfigMap["IssuerOrgName"] = issuerOrgName

	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: currentConfigMap,
	}

	if _, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{}); apiErrors.IsNotFound(err) { 
		// create configMap if not already present
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	} else {
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	}
	if err != nil {
		return errors.Wrap(err, "failed saving issuer metadata to kubernetes")
	}
	// _, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed saving issuer metadata to kubernetes")
	// }
	return nil
}

func getConfigMap() map[string]string {
	log.Info("This function in sentry gets ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		log.Fatalf("could not get kubernetes client, err: %s", err)
	}
	namespace := getNamespace()
	configMapName := "dapr-config-map"
	if _, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{}); apiErrors.IsNotFound(err) {
		// if map was not found create one
		log.Infof("in here?")
		configMap := &v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: make(map[string]string),
		}
		confMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			log.Infof("failed to create config map, err: %s", err)
		}
		return confMap.Data
	} else {
		log.Infof("map was found")
		configMap, _ := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if configMap.Data == nil {
			return make(map[string]string)
		}
		return configMap.Data
	}
}