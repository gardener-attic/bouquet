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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sort"
)

type SortOrder []string

var InstallOrder SortOrder = []string{
	"Namespace",
	"ResourceQuota",
	"LimitRange",
	"Secret",
	"ConfigMap",
	"StorageClass",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"ServiceAccount",
	"CustomResourceDefinition",
	"ClusterRole",
	"ClusterRoleBinding",
	"Role",
	"RoleBinding",
	"Service",
	"DaemonSet",
	"Pod",
	"ReplicationController",
	"ReplicaSet",
	"Deployment",
	"StatefulSet",
	"Job",
	"CronJob",
	"Ingress",
	"APIService",
}

var UninstallOrder SortOrder = []string{
	"APIService",
	"Ingress",
	"Service",
	"CronJob",
	"Job",
	"StatefulSet",
	"Deployment",
	"ReplicaSet",
	"ReplicationController",
	"Pod",
	"DaemonSet",
	"RoleBinding",
	"Role",
	"ClusterRoleBinding",
	"ClusterRole",
	"CustomResourceDefinition",
	"ServiceAccount",
	"PersistentVolumeClaim",
	"PersistentVolume",
	"StorageClass",
	"ConfigMap",
	"Secret",
	"LimitRange",
	"ResourceQuota",
	"Namespace",
}

func SortObjects(objects []*unstructured.Unstructured, order SortOrder) {
	sorter := newKindSorter(objects, order)
	sort.Sort(sorter)
}

func newKindSorter(objects []*unstructured.Unstructured, order SortOrder) *kindSorter {
	ordering := make(map[string]int, len(order))
	for v, k := range order {
		ordering[k] = v
	}

	return &kindSorter{
		objects:  objects,
		ordering: ordering,
	}
}

type kindSorter struct {
	objects  []*unstructured.Unstructured
	ordering map[string]int
}

func (k *kindSorter) Len() int {
	return len(k.objects)
}

func (k *kindSorter) Less(i, j int) bool {
	a := k.objects[i]
	b := k.objects[j]
	first, aok := k.ordering[a.GetKind()]
	second, bok := k.ordering[b.GetKind()]

	if first == second {
		if !aok && !bok && a.GetKind() != b.GetKind() {
			return a.GetKind() < b.GetKind()
		}
		return a.GetName() < b.GetName()
	}

	if !aok {
		return false
	}
	if !bok {
		return true
	}
	return first < second
}

func (k *kindSorter) Swap(i, j int) {
	k.objects[i], k.objects[j] = k.objects[j], k.objects[i]
}
