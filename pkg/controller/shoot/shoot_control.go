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

package shoot

import (
	"encoding/json"
	"fmt"
	"github.com/blang/semver"
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
	"github.com/gardener/bouquet/pkg/controller/common"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	anyVersion = semver.Range(func(_ semver.Version) bool { return true })
)

func (c *Controller) extractAddonManifests(shoot *gardenv1beta1.Shoot) ([]*v1alpha1.AddonManifest, error) {
	jsonAddonNames := shoot.Annotations[v1alpha1.AddonAnnotation]
	var addonNames []string
	if err := json.Unmarshal([]byte(jsonAddonNames), &addonNames); err != nil {
		return nil, err
	}

	if addonNameSet := sets.NewString(addonNames...); addonNameSet.Len() != len(addonNames) {
		return nil, fmt.Errorf("duplicate addons specified: %s", addonNames)
	}

	var manifests []*v1alpha1.AddonManifest
	for _, addonName := range addonNames {
		clusterManifests, err := c.addonManifestsLister.List(labels.Everything())
		if err != nil {
			return nil, err
		}

		manifest, err := common.FindManifestForRange(clusterManifests, addonName, anyVersion)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func getAddonInstanceName(shoot *gardenv1beta1.Shoot, manifest *v1alpha1.AddonManifest) string {
	name, _ := manifest.NameAndVersion()
	return fmt.Sprintf("%s-%s", shoot.Name, name)
}

func (c *Controller) ensureAddons(shoot *gardenv1beta1.Shoot, manifests []*v1alpha1.AddonManifest) error {
	var result error

	for _, manifest := range manifests {
		manifestName, manifestVersion := manifest.NameAndVersion()
		instance := &v1alpha1.AddonInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       "AddonInstance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: shoot.Namespace,
				Name:      getAddonInstanceName(shoot, manifest),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(shoot, gardenv1beta1.SchemeGroupVersion.WithKind("Shoot")),
				},
				Finalizers: []string{v1alpha1.BouquetName},
			},
			Spec: v1alpha1.AddonInstanceSpec{
				Manifest: v1alpha1.ManifestRef{
					Namespace: manifest.Namespace,
					Name:      manifestName,
					Version:   manifestVersion.String(),
				},
				Target: v1alpha1.Target{
					Shoot: shoot.Name,
				},
			},
		}

		_, err := c.bouquetclientset.GardenV1alpha1().AddonInstances(shoot.Namespace).Create(instance)
		if err != nil && !errors.IsAlreadyExists(err) {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func (c *Controller) reconcile(shoot *gardenv1beta1.Shoot) error {
	if shoot.DeletionTimestamp != nil {
		return nil
	}

	manifests, err := c.extractAddonManifests(shoot)
	if err != nil {
		c.log.Errorf("Could not resolve addon manifests for shoot %s/%s: %v",
			shoot.Namespace, shoot.Name, err)
		return err
	}

	if err := c.ensureAddons(shoot, manifests); err != nil {
		c.log.Errorf("Could not ensure addons for shoot %s/%s: %v",
			shoot.Namespace, shoot.Name, err)
		return err
	}

	return nil
}
