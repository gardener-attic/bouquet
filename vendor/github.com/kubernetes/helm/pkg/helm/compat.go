/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package helm

import (
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// These APIs are a temporary abstraction layer that captures the interaction between the current cmd/helm and old
// pkg/helm implementations. Post refactor the cmd/helm package will use the APIs exposed on helm.Client directly.

var Config struct {
	ServAddr string
}

// Soon to be deprecated helm ListReleases API.
func ListReleases(limit int, offset string, sort rls.ListSort_SortBy, order rls.ListSort_SortOrder, filter string) (*rls.ListReleasesResponse, error) {
	opts := []ReleaseListOption{
		ReleaseListLimit(limit),
		ReleaseListOffset(offset),
		ReleaseListFilter(filter),
		ReleaseListSort(int32(sort)),
		ReleaseListOrder(int32(order)),
	}
	return NewClient(HelmHost(Config.ServAddr)).ListReleases(opts...)
}

// Soon to be deprecated helm GetReleaseStatus API.
func GetReleaseStatus(rlsName string) (*rls.GetReleaseStatusResponse, error) {
	return NewClient(HelmHost(Config.ServAddr)).ReleaseStatus(rlsName)
}

// Soon to be deprecated helm GetReleaseContent API.
func GetReleaseContent(rlsName string) (*rls.GetReleaseContentResponse, error) {
	return NewClient(HelmHost(Config.ServAddr)).ReleaseContent(rlsName)
}

// Soon to be deprecated helm UpdateRelease API.
func UpdateRelease(rlsName string) (*rls.UpdateReleaseResponse, error) {
	return NewClient(HelmHost(Config.ServAddr)).UpdateRelease(rlsName)
}

// Soon to be deprecated helm InstallRelease API.
func InstallRelease(vals []byte, rlsName, chStr string, dryRun bool) (*rls.InstallReleaseResponse, error) {
	client := NewClient(HelmHost(Config.ServAddr))
	if dryRun {
		client.Option(DryRun())
	}
	return client.InstallRelease(chStr, ValueOverrides(vals), ReleaseName(rlsName))
}

// Soon to be deprecated helm UninstallRelease API.
func UninstallRelease(rlsName string, dryRun bool) (*rls.UninstallReleaseResponse, error) {
	client := NewClient(HelmHost(Config.ServAddr))
	if dryRun {
		client.Option(DryRun())
	}
	return client.DeleteRelease(rlsName)
}
