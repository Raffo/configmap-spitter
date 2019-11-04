package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := kingpin.Flag("kubeconfig", "path to kubeconfig file.").String()
	configmapNames := kingpin.Flag("configmaps", "List of configmaps to write").Required().Strings()
	namespace := kingpin.Flag("namespace", "Namespace for which the configmaps will be combined").Required().String()
	pathToWriteTo := kingpin.Flag("write-path", "Path to write to").Required().String()
	kingpin.Parse()

	clientConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logrus.Fatalf("cannot build config: %v", err)
	}

	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		logrus.Fatalf("cannot build kubeclient: %v", err)
	}

	for _, ns := range *configmapNames {
		configMap, err := client.CoreV1().ConfigMaps(*namespace).Get(ns, v1.GetOptions{})
		if err != nil {
			panic(err) // let it crash
		}

		for fileName, data := range configMap.Data {
			f, err := os.Create(fmt.Sprintf("%s/%s", *pathToWriteTo, fileName))
			if err != nil {
				panic(err)
			}
			_, err = f.WriteString(data)
			if err != nil {
				panic(err)
			}
		}
	}
}
