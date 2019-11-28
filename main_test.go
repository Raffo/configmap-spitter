package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func OldTest(t *testing.T) {
	client := fake.NewSimpleClientset()
	configmapNames := []string{"foo", "bar"}
	namespace := "default"
	pathToWriteTo := "/output"

	fakeFS := &fakeFS{file: []*fakeFile{}}

	err := copyConfigmaps(client, fakeFS, configmapNames, namespace, pathToWriteTo)
	assert.Error(t, err)
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
		})
	}
}
