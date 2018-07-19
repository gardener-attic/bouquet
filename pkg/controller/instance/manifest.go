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
	"encoding/json"
	"fmt"
	"github.com/blang/semver"
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
	"github.com/gardener/bouquet/pkg/controller/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

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

func (c *Controller) resolveManifest(manifest *v1alpha1.AddonManifest) (*chart.Chart, error) {
	source := manifest.Spec
	if source.ConfigMap == "" {
		return nil, fmt.Errorf("no source for manifest %s/%s", manifest.Namespace, manifest.Name)
	}

	return c.resolveManifestConfigMap(manifest)
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

func (c *Controller) resolveManifestConfigMap(manifest *v1alpha1.AddonManifest) (*chart.Chart, error) {
	configMap, err := c.kubeclientset.
		CoreV1().
		ConfigMaps(manifest.Namespace).
		Get(manifest.Spec.ConfigMap, metav1.GetOptions{})
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

	templates := make([]*chart.Template, 0, len(configMap.Data))
	for name, data := range configMap.Data {
		templates = append(templates, &chart.Template{Name: name, Data: []byte(data)})
	}

	return &chart.Chart{
		Metadata:  metadata,
		Values:    config,
		Templates: templates,
	}, nil
}
