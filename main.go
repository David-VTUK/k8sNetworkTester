package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
	"strings"
)

// Channel buffer size
const (
	defaultKubeconfig = "~/.kube/config"
	burst             = 50
	qps               = 25
)

func main() {
	kubeconfig, err := getConfig()
	if err != nil {
		handleError(err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		handleError(err)
	}

	// Increase the Burst and QOS values
	config.Burst = burst
	config.QPS = qps

	// Build client from  config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		handleError(err)
	}

	ctx := context.Background()

	// numberOfContainers, err := getTotalNumberOfContainers(ctx, clientset)

	if err != nil {
		handleError(err)
	}

	getTotalNumberOfNodes(ctx, clientset)

}

func getTotalNumberOfNodes(ctx context.Context, clientset *kubernetes.Clientset) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err != nil {
		handleError(err)
	}

	for _, node := range nodes.Items {
		if node.Labels["node-role.kubernetes.io/worker"] == "true" {
			log.Infof("Found Worker")
		}
	}
}

/*
func getTotalNumberOfContainers(ctx context.Context, clientSet *kubernetes.Clientset) (int, error) {

	numberofcontainers := 0
	pods, err := clientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	for _, pod := range pods.Items {
		numberofcontainers += len(pod.Spec.Containers)
	}
	return numberofcontainers, nil
}
*/

func getConfig() (string, error) {
	var filename string
	var err error
	kubeconfigFlag := flag.String("kubeconfig", "", "path to the kubeconfig file")
	flag.Parse()

	filename = *kubeconfigFlag
	if filename == "" {
		filename = defaultKubeconfig
	}
	filename, err = homeDir(filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filename); err != nil {
		return "", err
	}

	return filename, nil
}

func homeDir(filename string) (string, error) {
	if strings.Contains(filename, "~/") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		filename = strings.Replace(filename, "~/", "", 1)
		filename = path.Join(homedir, filename)
	}
	return filename, nil
}

func handleError(err error) {
	panic(err.Error())
}
