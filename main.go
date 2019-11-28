package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type fileSystem interface {
	Create(name string) (file, error)
}

type realFS struct{}

func (r *realFS) Create(name string) (file, error) {
	return os.Create(name)
}

type file interface {
	WriteString(s string) (n int, err error)
}

type realFile struct{}

func (f *realFile) WriteString(s string) (n int, err error) {
	return f.WriteString(s)
}

type config struct {
	interval       time.Duration
	stopChannel    chan int
	client         kubernetes.Interface
	realFS         fileSystem
	configmapNames []string
	namespace      string
	pathToWriteTo  string
	loop           bool
}

func copyConfigmaps(client kubernetes.Interface, os fileSystem, configmapNames []string, namespace, pathToWriteTo string) error {
	for _, ns := range configmapNames {
		configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ns, v1.GetOptions{})
		if err != nil {
			return err
		}

		for fileName, data := range configMap.Data {
			f, err := os.Create(fmt.Sprintf("%s/%s", pathToWriteTo, fileName))
			if err != nil {
				return err
			}
			_, err = f.WriteString(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func run(c *config, f func(client kubernetes.Interface, os fileSystem, configmapNames []string, namespace, pathToWriteTo string) error) {
	for {
		select {
		case <-time.After(c.interval):
			err := f(c.client, c.realFS, c.configmapNames, c.namespace, c.pathToWriteTo)
			if err != nil {
				logrus.Fatalf("error copying configmaps: %v", err)
			}
			if !c.loop {
				logrus.Infoln("no loop requested, exiting.")
				return
			}
		case <-c.stopChannel:
			logrus.Infoln("stop requested, exiting.")
			return
		}
	}
}

func main() {
	kubeconfig := kingpin.Flag("kubeconfig", "path to kubeconfig file.").String()
	configmapNames := kingpin.Flag("configmaps", "List of configmaps to write").Required().Strings()
	namespace := kingpin.Flag("namespace", "Namespace for which the configmaps will be combined").Required().String()
	pathToWriteTo := kingpin.Flag("write-path", "Path to write to").Required().String()
	loop := kingpin.Flag("loop", "Run in a never terminating loop").Default("false").Bool()
	interval := kingpin.Flag("interval", "Interval for which the configmaps will be copied").Duration()
	kingpin.Parse()

	clientConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logrus.Fatalf("cannot build config: %v", err)
	}

	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		logrus.Fatalf("cannot build kubeclient: %v", err)
	}

	stopChannel := make(chan int)

	realFS := &realFS{}

	run(&config{
		interval:       *interval,
		stopChannel:    stopChannel,
		client:         client,
		realFS:         realFS,
		configmapNames: *configmapNames,
		namespace:      *namespace,
		pathToWriteTo:  *pathToWriteTo,
		loop:           *loop,
	}, copyConfigmaps)

}
