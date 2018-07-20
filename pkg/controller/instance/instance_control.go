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
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
	"github.com/hashicorp/go-multierror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

const (
	kubeConfigKey = "kubeconfig"
)

type ResourceFactory func(resource *metav1.APIResource, namespace string) (dynamic.ResourceInterface, error)

// TODO: caching of shoot clients / rest mappers
func (c *Controller) targetFromShoot(shootNamespace, shootName string) (dynamic.Interface, meta.RESTMapper, error) {
	shoot, err := c.gardenclientset.GardenV1beta1().Shoots(shootNamespace).Get(shootName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	// TODO: Improved checking of 'readiness' of Shoot
	cfg, err := c.shootRestConfig(shoot)
	if err != nil {
		return nil, nil, err
	}

	dynamicInterface, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(kubeClient.Discovery())
	if err != nil {
		return nil, nil, err
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	return dynamicInterface, restMapper, nil
}

// TODO: URL file / tar.gz source
func (c *Controller) targetFromInstance(instance *v1alpha1.AddonInstance) (dynamic.Interface, meta.RESTMapper, error) {
	if instance.Spec.Target.Shoot == "" {
		return c.dynamicclient, c.restMapper, nil
	}

	return c.targetFromShoot(instance.Namespace, instance.Spec.Target.Shoot)
}

func dynamicResourceInterfaceFor(
	dynamicInterface dynamic.Interface,
	restMapper meta.RESTMapper,
	object *unstructured.Unstructured,
) (dynamic.ResourceInterface, error) {

	gvk := object.GetObjectKind().GroupVersionKind()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	if mapping.Scope != meta.RESTScopeNamespace {
		return dynamicInterface.Resource(mapping.Resource), nil
	}

	namespace := object.GetNamespace()
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return dynamicInterface.Resource(mapping.Resource).Namespace(namespace), nil
}

func (c *Controller) ensureAddonInstance(instance *v1alpha1.AddonInstance, objects []*unstructured.Unstructured) error {
	dynamicInterface, restMapper, err := c.targetFromInstance(instance)
	if err != nil {
		return err
	}

	var result error
	for _, object := range objects {
		resourceInterface, err := dynamicResourceInterfaceFor(dynamicInterface, restMapper, object)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		// TODO: Other modes like K8S's AddonManger's Reconcile + Fire and Forget mode
		if _, err := resourceInterface.Create(object); err != nil && !apierrors.IsAlreadyExists(err) {
			result = multierror.Append(result, err)
			continue
		}
	}

	return result
}

func (c *Controller) deleteAddonInstance(instance *v1alpha1.AddonInstance, objects []*unstructured.Unstructured) error {
	// TODO: check case where shoot has already been disassociated from seed but objects were created
	dynamicInterface, restMapper, err := c.targetFromInstance(instance)
	if err != nil {
		return err
	}

	var result error
	for _, object := range objects {
		resourceInterface, err := dynamicResourceInterfaceFor(dynamicInterface, restMapper, object)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		if err := resourceInterface.Delete(object.GetName(), nil); err != nil && !apierrors.IsNotFound(err) {
			result = multierror.Append(result, err)
			continue
		}
	}

	return result
}

func (c *Controller) reconcile(instance *v1alpha1.AddonInstance) error {
	ref := instance.Spec.Manifest
	manifest, err := c.findManifestByRef(ref)
	if err != nil {
		c.log.Errorf("Could not find manifest: %v", err)
		return err
	}

	src, err := c.resolveManifest(manifest)
	if err != nil {
		c.log.Errorf("Could not resolve manifest %q: %v", manifest.Name, err)
		return err
	}

	objects, err := src.Apply(instance)
	if err != nil {
		c.log.Errorf("Could not apply instance to manifest: %v", err)
		return err
	}

	if instance.DeletionTimestamp != nil {
		if !sets.NewString(instance.Finalizers...).Has(v1alpha1.BouquetName) {
			return nil
		}

		// TODO: Cleanup should not happen on an instance -> Move logic
		// to sth like addon instance object.
		if err := c.deleteAddonInstance(instance, objects); err != nil {
			c.log.Errorf("Could not delete addon instance: %v", err)
			return err
		}

		instanceFinalizers := sets.NewString(instance.Finalizers...)
		instanceFinalizers.Delete(v1alpha1.BouquetName)
		instance.Finalizers = instanceFinalizers.UnsortedList()
		if _, err := c.bouquetclientset.GardenV1alpha1().AddonInstances(instance.Namespace).Update(instance); err != nil && !apierrors.IsNotFound(err) {
			c.log.Errorf("Could not remove finalizer from addon instance %s/%s: %v",
				instance.Namespace, instance.Name, err)
			return err
		}

		c.log.Infof("Successfully cleaned up %s/%s", manifest.Name, instance.Name)
		return nil
	}

	if err := c.ensureAddonInstance(instance, objects); err != nil {
		c.log.Errorf("Could not ensure addons: %v", err)
		return err
	}

	return nil
}
