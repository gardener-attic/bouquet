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

package common

import (
	"fmt"
	"github.com/blang/semver"
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
)

// TODO: Remove auto-latest behavior
func FindManifestForRange(manifests []*v1alpha1.AddonManifest, name string, targetRange semver.Range) (*v1alpha1.AddonManifest, error) {
	var version *semver.Version
	var manifest *v1alpha1.AddonManifest

	for _, curManifest := range manifests {
		curName, curVersion := curManifest.NameAndVersion()

		if curName == name && targetRange(curVersion) && (version == nil || curVersion.GT(*version)) {
			version = &curVersion
			manifest = curManifest
		}
	}

	if nil == manifest {
		return nil, fmt.Errorf("no matching manifest for %s", name)
	}
	return manifest, nil
}
