package process

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func restartPods() error {
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
