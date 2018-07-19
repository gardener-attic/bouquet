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

package v1alpha1

import (
	"github.com/blang/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"regexp"
)

var (
	versionNameRegex = regexp.MustCompile("^(.*)-(\\d+\\.\\d+\\.\\d+(-.*)*)$")
)

const (
	// BouquetName is the value in a Bouquet resource's `.metadata.finalizers[]` array on which
	// Bouquet will react when performing a delete request on a resource.
	BouquetName = "bouquet"

	AddonAnnotation = "gardenextensions.sapcloud.io/addons"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddonManifestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AddonManifest `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddonInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AddonInstance `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonManifest is a definition of an addon that can be instantiated.
type AddonManifest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonManifestSpec `json:"spec,omitempty"`
}

// TODO: Validate CRs with Open API schema so this never produces 'malformed' results
func (in *AddonManifest) NameAndVersion() (string, semver.Version) {
	matches := versionNameRegex.FindStringSubmatch(in.Name)
	if matches == nil {
		return "", semver.Version{}
	}

	name := matches[1]
	version, err := semver.Parse(matches[2])
	if err != nil {
		return name, semver.Version{}
	}

	return name, version
}

type AddonManifestSpec struct {
	ConfigMap    string                 `json:"configMap"`
	Values       map[string]interface{} `json:"values,omitempty"`
	Dependencies map[string]string      `json:"dependencies,omitempty"`
}

func (in *AddonManifestSpec) DeepCopyInto(out *AddonManifestSpec) {
	*out = *in
	out.Values = runtime.DeepCopyJSON(in.Values)
	out.Dependencies = make(map[string]string, len(in.Dependencies))
	for k, v := range in.Dependencies {
		out.Dependencies[k] = v
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonInstance is an instantiated AddonManifest (i.e. a templated and bound AddonManifest).
type AddonInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonInstanceSpec   `json:"spec,omitempty"`
	Status AddonInstanceStatus `json:"status,omitempty"`
}

type AddonInstanceSpec struct {
	Target   Target      `json:"target,omitempty"`
	Manifest ManifestRef `json:"manifest,omitempty"`
	// TODO: Respect dependency order, dependency resolution (building Graphs, detecting problems?)
	Values map[string]interface{} `json:"values,omitempty"`
}

type AddonInstanceStatus struct {
	// TODO: Use as 'cache' for successfully templated objects
	Objects []map[string]interface{} `json:"objects"`
}

func (in *AddonInstanceSpec) DeepCopyInto(out *AddonInstanceSpec) {
	*out = *in
	out.Values = runtime.DeepCopyJSON(in.Values)
}

func (in *AddonInstanceStatus) DeepCopyInto(out *AddonInstanceStatus) {
	*out = *in
	if in.Objects != nil {
		out.Objects = make([]map[string]interface{}, 0, len(in.Objects))
		for _, object := range in.Objects {
			out.Objects = append(out.Objects, runtime.DeepCopyJSON(object))
		}
	}
}

type ManifestRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// TODO: Should 'Shoot' be the only target ever possible (i.e. prohibit local use)?
type Target struct {
	Shoot string `json:"shoot,omitempty"`
}
