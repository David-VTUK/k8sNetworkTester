package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
	"strings"
	"time"
)

// Channel buffer size
const (
	defaultKubeconfig = "~/.kube/config"
	burst             = 50
	qps               = 25
)

var nodeList []string

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
	generateWorkloads(ctx, clientset)

}

func getTotalNumberOfNodes(ctx context.Context, clientset *kubernetes.Clientset) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err != nil {
		handleError(err)
	}

	for _, node := range nodes.Items {
		if node.Labels["node-role.kubernetes.io/worker"] == "true" {
			log.Infof("Found Worker %s", node.Name)
			nodeList = append(nodeList, node.Name)
		}
	}
}

func generateWorkloads(ctx context.Context, clientset *kubernetes.Clientset) {
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "connectivity-checker",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "connectivity-checker"},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "connectivity-checker"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.21.3",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	daemonsetClient := clientset.AppsV1().DaemonSets(apiv1.NamespaceDefault)

	log.Info("Creating Daemonset")
	result, err := daemonsetClient.Create(context.TODO(), daemonset, metav1.CreateOptions{})
	if err != nil {
		handleError(err)
	}

	log.Infof("Created Daemonset %s", result.Name)

	for i := 0; i < 30; i++ {
		log.Info(daemonset.Status)
		time.Sleep(2 * time.Second)
	}
}

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
