package configmap

import (
	"context"
	// "os"

	"github.com/pkg/errors"
	"github.com/dapr/dapr/pkg/sentry/kubernetes"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/dapr/kit/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapName = "dapr-config-map"
	defaultSecretNamespace = "default"
)

var log = logger.NewLogger("dapr.sentry")

// Gets namespace for the ConfigMap
func getNamespace() string {
	// namespace := os.Getenv("NAMESPACE")
	// if namespace == "" {
	// 	namespace = defaultSecretNamespace
	// }
	namespace := "dapr-system"
	return namespace
}

func WriteToConfigMap(key string, value string) error {
	log.Info("This function registers an action to ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		return err
	}
	namespace := getNamespace()
	currentConfigMap := getConfigMap() // get config map
	currentConfigMap[key] = value // add action id field

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
	log.Infof("created configmap update object")
	if _, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{}); apiErrors.IsNotFound(err) { 
		// create configMap if not already present
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	} else {
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	}
	if err != nil {
		return errors.Wrap(err, "failed to register action id to kubernetes")
	}
	log.Infof("successfully registered action to configmap")

	ReadKeyFromConfigMap(key)

	return nil
}

// check if some action is already present in key-value store
func ReadKeyFromConfigMap(key string) string {
	configMap := getConfigMap()
	val := configMap[key]
	return val
}

// gets configmap from kubernetes
func getConfigMap() map[string]string {
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		log.Fatalf("could not get kubernetes client, err: %s", err)
	}

	namespace := getNamespace()
	if _, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{}); apiErrors.IsNotFound(err) {
		// if map was not found create one
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
		log.Infof("configmap not found, creating...")
		newConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			log.Infof("failed to create config map, err: %s", err)
		}
		return newConfigMap.Data
	} else {
		configMap, _ := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		return configMap.Data
	}
}

