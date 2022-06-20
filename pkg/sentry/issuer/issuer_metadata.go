package issuer

import (
	"context"
	"os"

	"github.com/dapr/dapr/pkg/sentry/kubernetes"
	"github.com/dapr/kit/logger"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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

func WriteIssuerMetadataToConfigMap(issuerOrgName string) error {
	log.Info("This function writes a value to ConfigMap")
	kubeClient, err := kubernetes.GetClient()
	if err != nil {
		return err
	}
	namespace := getNamespace()
	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dapr-config-map",
			Namespace: namespace,
		},
		Data: map[string]string{
			"IssuerOrgName": issuerOrgName,
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed saving issuer metadata to kubernetes")
	}
	return nil
}