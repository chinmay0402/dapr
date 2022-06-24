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

func restart() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("could not get config, err: %s", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})

	// restart sentry first
	data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().Deployments("dapr-system").Patch(context.Background(), "dapr-sentry", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		log.Fatalf("couldn't restart sentry, err: %s", err)
	}
	log.Infof("sentry restart successful!")

	data = fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().Deployments("dapr-system").Patch(context.Background(), "dapr-operator", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		log.Fatalf("couldn't restart operator, err: %s", err)
	}
	log.Infof("operator restart successful!")

	data = fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
	_, err = clientset.AppsV1().StatefulSets("dapr-system").Patch(context.Background(), "dapr-placement-server", types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		log.Fatalf("couldn't restart statefulset placement server, err: %s", err)
	}
	log.Infof("statefulset placement server restart successful!")
	
	// restart application pods
	for _, item := range deployments.Items {

		deploymentNamespace := item.GetObjectMeta().GetNamespace()
		deploymentName := item.GetObjectMeta().GetName()
		if deploymentNamespace != "default" {
			continue
		}
		log.Infof("deployment namespace: %s, deployment name: %s", deploymentNamespace, deploymentName)

		data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}},"strategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":"%s","maxSurge": "%s"}}}`, time.Now().String(), "25%", "25%")
		_, err = clientset.AppsV1().Deployments(deploymentNamespace).Patch(context.Background(), deploymentName, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
		if err != nil {
			log.Fatalf("couldn't restart deployment, err: %s", err)
		}
		log.Infof("restart successful!")
	}
}
