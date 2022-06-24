package monitor

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/dapr/kit/logger"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dapr/dapr/pkg/health"
	"github.com/dapr/dapr/pkg/monitor/process"
)

var log = logger.NewLogger("dapr.monitor")

const (
	healthzPort = 8080
	daprAnnotationsMonitorEnabledKey = "dapr.io/enable-monitor"
	daprNamespace = "dapr-system"
)

type Monitor interface {
	Run(ctx context.Context)
}

type monitor struct {
	ctx context.Context 
	instanceName string
}

// returns a new monitor instance
func NewMonitor() Monitor {
	m := &monitor{
		instanceName: "monitor-instance",
	}
	log.Info("instance: %s", m.instanceName)

	return m
}

func (m *monitor) Run(ctx context.Context) {
	defer runtimeutil.HandleCrash()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	m.ctx = ctx
	go func() {
		<-ctx.Done()
		log.Infof("Dapr Monitor is shutting down")
	}()

	log.Infof("Dapr Monitor updated version started")

	go func() {
		healthzServer := health.NewServer(log)
		healthzServer.Ready()

		healthzErr := healthzServer.Run(ctx, healthzPort)
		if healthzErr != nil {
			log.Fatalf("failed to start healthz server: %s", healthzErr)
		}
	}()

	getLogs(ctx)
}

// Periodically fetches logs from control plane service pods and annotated application pods
func getLogs(ctx context.Context) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("could not get config, err: %s", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("could not get access to k8s, err: %s", err)
	}
	log.Info("getting logs every 30 seconds")
	for {
		// get list of all pods in all namespaces
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{}) 
		if err != nil {
			log.Fatalf("could not get pods, err: %s", err)
		}

		for _, pod := range pods.Items {
			if pod.Namespace == "kube-node-lease" || pod.Namespace == "kube-public" || pod.Namespace == "kube-system" {
				// ignore k8s specific namespaces
				continue
			}
			if pod.Namespace == daprNamespace && strings.Contains(pod.Name, "dapr-monitor") {
				// don't monitor logs of dapr-monitor itself
				continue
			}
			containsSidecar := false
			for container := range pod.Spec.Containers {
				if pod.Spec.Containers[container].Name == "daprd" {
					containsSidecar = true
					break
				}
			}
			if containsSidecar == false && pod.Namespace != daprNamespace {
				continue
			}
			// get annotations here, get logs if annotations are valid
			annotations := getPodAnnotations(clientset, pod.Namespace, pod.Name)
			value, isPresent := annotations[daprAnnotationsMonitorEnabledKey]
			if pod.Namespace != daprNamespace && (isPresent == false || value == "false") {
				continue
			}

			logs := getPodLogs(clientset, pod.Namespace, pod.Name, ctx) // get logs of the pod
			// process logs
			process.ProcessLogs(logs)
		}
		time.Sleep(30 * time.Second)
	}

}

// Fetches logs from a pod provided the pod
func getPodLogs(clientset *kubernetes.Clientset, podNamespace string, podName string, ctx context.Context) string {
	f := func(s int64) *int64 {
        return &s
    }
	podLogOpts := v1.PodLogOptions{
		SinceSeconds: f(60),
	} // TODO: look into pod options: Container, Follow, SinceSeconds
	if podNamespace != daprNamespace {
		// get logs for daprd container only since app specific errors cannot be handled anyways
		podLogOpts.Container = "daprd"
	}
	req := clientset.CoreV1().Pods(podNamespace).GetLogs(podName, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Fatalf("error in opening stream, err: %s", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Fatalf("error in copying logs from podLogs to buf, err: %s", err)
	}
	str := buf.String()
	// log.Infof("%s logs => %s", podName, str)
	return str
}

// Fetches annotations for given pod
func getPodAnnotations(clientset *kubernetes.Clientset, podNamespace string, podName string) map[string]string {
	pod, err := clientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("could not get pod data, err: %s", err)
	}
	return pod.GetAnnotations()
}
