/*
Copyright YEAR (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
     http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/blang/semver"
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
	"github.com/gardener/bouquet/pkg/controller/common"
	"github.com/golang/protobuf/ptypes/any"
	helmEngine "github.com/kubernetes/helm/pkg/engine"
	"io"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"strings"
)

var (
	engine = helmEngine.New()
)

type Source interface {
	Apply(instance *v1alpha1.AddonInstance) ([]*unstructured.Unstructured, error)
}

func ParseObjects(data []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(data))

	for {
		var decoded map[string]interface{}
		if err := decoder.Decode(&decoded); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		if decoded == nil {
			continue
		}

		object := &unstructured.Unstructured{Object: decoded}
		objects = append(objects, object)
	}

	return objects, nil
}

func ParseMappedFileContents(mappedContents map[string]string) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	for _, content := range mappedContents {
		newObjects, err := ParseObjects([]byte(content))
		if err != nil {
			return nil, err
		}

		objects = append(objects, newObjects...)
	}

	return objects, nil
}

type chartSource struct {
	chrt *chart.Chart
}

func (c *chartSource) Apply(instance *v1alpha1.AddonInstance) ([]*unstructured.Unstructured, error) {
	values, err := engine.Render(c.chrt, instance.Spec.Values)
	if err != nil {
		return nil, err
	}

	return ParseMappedFileContents(values)
}

type mappedFileSource struct {
	mappedFiles map[string]string
}

func (m *mappedFileSource) Apply(_ *v1alpha1.AddonInstance) ([]*unstructured.Unstructured, error) {
	return ParseMappedFileContents(m.mappedFiles)
}

func (c *Controller) findManifestByRef(ref v1alpha1.ManifestRef) (*v1alpha1.AddonManifest, error) {
	targetRange, err := semver.ParseRange(ref.Version)
	if err != nil {
		return nil, err
	}

	addonManifests, err := c.addonManifestsLister.AddonManifests(ref.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return common.FindManifestForRange(addonManifests, ref.Name, targetRange)
}

func (c *Controller) configMapFor(manifest *v1alpha1.AddonManifest) (*v1.ConfigMap, error) {
	return c.kubeclientset.
		CoreV1().
		ConfigMaps(manifest.Namespace).
		Get(manifest.Spec.ConfigMap, metav1.GetOptions{})
}

func (c *Controller) resolveManifest(manifest *v1alpha1.AddonManifest) (Source, error) {
	source := manifest.Spec

	if source.ConfigMap != "" {
		return c.resolveConfigMap(manifest)
	}
	if source.ConfigMapHelm != "" {
		return c.resolveConfigMapHelm(manifest)
	}

	return nil, fmt.Errorf("no source for manifest %s/%s", manifest.Namespace, manifest.Name)
}

func (c *Controller) resolveConfigMap(manifest *v1alpha1.AddonManifest) (Source, error) {
	configMap, err := c.configMapFor(manifest)
	if err != nil {
		return nil, err
	}

	return &mappedFileSource{configMap.Data}, nil
}

func chartMetadata(manifest *v1alpha1.AddonManifest) (*chart.Metadata, error) {
	name, version := manifest.NameAndVersion()

	return &chart.Metadata{
		Name:    name,
		Version: version.String(),
	}, nil
}

func chartConfig(manifest *v1alpha1.AddonManifest) (*chart.Config, error) {
	data, err := json.Marshal(manifest.Spec.Values)
	if err != nil {
		return nil, err
	}

	return &chart.Config{Raw: string(data)}, nil
}

func (c *Controller) resolveConfigMapHelm(manifest *v1alpha1.AddonManifest) (Source, error) {
	configMap, err := c.configMapFor(manifest)
	if err != nil {
		return nil, err
	}

	metadata, err := chartMetadata(manifest)
	if err != nil {
		return nil, err
	}

	config, err := chartConfig(manifest)
	if err != nil {
		return nil, err
	}

	var templates []*chart.Template
	var files []*any.Any
	for name, data := range configMap.Data {
		if strings.HasSuffix(name, ".tmpl") {
			templates = append(templates, &chart.Template{Name: name, Data: []byte(data)})
		} else {
			files = append(files, &any.Any{TypeUrl: name, Value: []byte(data)})
		}
	}

	chrt := &chart.Chart{
		Metadata:  metadata,
		Values:    config,
		Templates: templates,
		Files:     files,
	}

	return &chartSource{chrt}, nil
}
