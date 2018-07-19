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
	"errors"
	"fmt"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	errNoKubeConfig = errors.New("no kube config found")
)

func newRESTConfigFromSecret(secret *v1.Secret) (*rest.Config, error) {
	kubeConfig, ok := secret.Data[kubeConfigKey]
	if !ok {
		return nil, errNoKubeConfig
	}

	return clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
}

func (c *Controller) seedRESTConfig(seed *gardenv1beta1.Seed) (*rest.Config, error) {
	secretRef := seed.Spec.SecretRef
	secret, err := c.kubeclientset.CoreV1().Secrets(secretRef.Namespace).Get(secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return newRESTConfigFromSecret(secret)
}

func (c *Controller) seedClient(seed *gardenv1beta1.Seed) (kubernetes.Interface, error) {
	config, err := c.seedRESTConfig(seed)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func (c *Controller) shootRestConfig(shoot *gardenv1beta1.Shoot) (*rest.Config, error) {
	seedName := shoot.Status.Seed
	if seedName == "" {
		return nil, fmt.Errorf("shoot %s/%s is not yet associated to a seed",
			shoot.Namespace, shoot.Name)
	}

	seed, err := c.gardenclientset.GardenV1beta1().Seeds().Get(seedName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	seedClient, err := c.seedClient(seed)
	if err != nil {
		return nil, err
	}

	namespace := shootpkg.ComputeTechnicalID("garden-dev", shoot)
	secret, err := seedClient.CoreV1().Secrets(namespace).Get(gardenv1beta1.GardenerName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return newRESTConfigFromSecret(secret)
}

func (c *Controller) shootClient(shoot *gardenv1beta1.Shoot) (kubernetes.Interface, error) {
	config, err := c.shootRestConfig(shoot)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
