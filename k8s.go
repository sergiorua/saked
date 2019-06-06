package main

import (
	"os"
	"log"

    "k8s.io/client-go/rest"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	v1 "k8s.io/api/core/v1"
)

func fileExists(name string) bool {
    if _, err := os.Stat(name); err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}

func connectK8s(kubeconfig string) (*kubernetes.Clientset, error) {
    var config *rest.Config
    var err error
    var clientset *kubernetes.Clientset

    if fileExists(kubeconfig) == false {
		config, err = rest.InClusterConfig()
		if err != nil {
			return clientset, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return clientset, err
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err.Error())
	}

    return clientset, err
}

func discoverK8sService(kubeconfig string, name string, namespace string) (v1.Service, error) {
    var s v1.Service

    clientset, err := connectK8s(kubeconfig)
    if err != nil {
        return s, err
    }

    services, err := clientset.CoreV1().Services("").List(metav1.ListOptions{})
    if err != nil {
        return s, err
    }

    for _, s = range services.Items {
        if s.ObjectMeta.Name == name && s.ObjectMeta.Namespace == namespace {
            return s, nil
        }
    }
    return s, err
}