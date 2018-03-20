package cmd

import (
	"os"
	"k8s.io/client-go/kubernetes"
	"flag"
	"path/filepath"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"log"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func K8SClient() (*kubernetes.Clientset, error) {
	config, err := K8SConfig()

	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

var cachedConfig *rest.Config
func K8SConfig() (*rest.Config, error) {
	if cachedConfig != nil {
		return cachedConfig, nil
	}

	var kubeconfig *string

	config, err := rest.InClusterConfig()

	if err != nil {
		log.Println("in cluster config failed, trying from local")
		if home := homeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	cachedConfig = config

	return config, nil
}
