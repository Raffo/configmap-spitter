package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeFS struct {
	file []*fakeFile // key: fileName, value: fileContent
}

type fakeFile struct {
	name    string
	content string
}

func (f *fakeFile) WriteString(s string) (n int, err error) {
	f.content = s
	return len(s), nil
}

func (r *fakeFS) Create(name string) (file, error) {
	f := &fakeFile{
		name: name,
	}
	r.file = append(r.file, f)
	return f, nil
}

func TestCopyConfigMap(tt *testing.T) {
	client := fake.NewSimpleClientset()
	fakeFS := &fakeFS{file: []*fakeFile{}}
	for _, test := range []struct {
		name          string
		configMaps    []string
		namespace     string
		pathToWriteTo string
		errorExpected bool
	}{
		{
			name:          "delete all namespaces",
			configMaps:    []string{"foo", "bar"},
			namespace:     "default",
			pathToWriteTo: "/foo",
			errorExpected: false,
		},
		{
			name:          "delete all namespaces",
			configMaps:    []string{"foo"},
			namespace:     "default",
			pathToWriteTo: "/foo",
			errorExpected: false,
		},
		{
			name:          "delete all namespaces",
			configMaps:    []string{},
			namespace:     "default",
			pathToWriteTo: "/foo",
			errorExpected: false,
		},
	} {
		tt.Run(test.name, func(t *testing.T) {
			for _, c := range test.configMaps {
				client.CoreV1().ConfigMaps("default").Create(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: c,
					},
					Data: map[string]string{
						c: c,
					},
				})
			}

			err := copyConfigmaps(client, fakeFS, test.configMaps, test.namespace, test.pathToWriteTo)
			if test.errorExpected {
				assert.Error(t, err)
			}
			assert.Len(t, fakeFS.file, len(test.configMaps))
			// check that the content matches
			for index, file := range fakeFS.file {
				assert.Equal(t, file.name, fmt.Sprintf("%s/%s", test.pathToWriteTo, test.configMaps[index]))
				assert.Equal(t, file.content, test.configMaps[index])
			}

			// reset the files for the next test run
			fakeFS.file = []*fakeFile{}
		})
	}
}

func TestRunWithLoopDisabled(t *testing.T) {
	client := fake.NewSimpleClientset()
	fakeFS := &fakeFS{file: []*fakeFile{}}
	stopChannel := make(chan int)
	count := 0
	run(&config{
		interval:       1 * time.Second,
		stopChannel:    stopChannel,
		client:         client,
		realFS:         fakeFS,
		configmapNames: []string{},
		namespace:      "default",
		pathToWriteTo:  "/foo",
		loop:           false,
	}, func(client kubernetes.Interface, os fileSystem, configmapNames []string, namespace, pathToWriteTo string) error {
		count++
		return nil
	})
}

func TestRunWithLoop(t *testing.T) {
	client := fake.NewSimpleClientset()
	fakeFS := &fakeFS{file: []*fakeFile{}}
	stopChannel := make(chan int)
	count := 0
	m := sync.Mutex{}
	go func() {

		for {
			select {
			case <-time.After(1 * time.Second):
				m.Lock()
				if count > 2 {
					stopChannel <- 1
					return
				}
				m.Unlock()
			}
		}
	}()
	run(&config{
		interval:       1 * time.Second,
		stopChannel:    stopChannel,
		client:         client,
		realFS:         fakeFS,
		configmapNames: []string{},
		namespace:      "default",
		pathToWriteTo:  "/foo",
		loop:           true,
	}, func(client kubernetes.Interface, os fileSystem, configmapNames []string, namespace, pathToWriteTo string) error {
		m.Lock()
		defer m.Unlock()
		count++
		return nil
	})

}
