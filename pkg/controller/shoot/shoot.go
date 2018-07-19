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
	"fmt"
	"github.com/gardener/bouquet/pkg/apis/garden/v1alpha1"
	clientset "github.com/gardener/bouquet/pkg/client/clientset/versioned"
	bouquetscheme "github.com/gardener/bouquet/pkg/client/clientset/versioned/scheme"
	v1alpha1informers "github.com/gardener/bouquet/pkg/client/informers/externalversions/garden/v1alpha1"
	listers "github.com/gardener/bouquet/pkg/client/listers/garden/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardenv1beta1informers "github.com/gardener/gardener/pkg/client/garden/informers/externalversions/garden/v1beta1"
	gardenv1beta1listers "github.com/gardener/gardener/pkg/client/garden/listers/garden/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

const (
	maxRetries = 15
)

type Controller struct {
	log *logrus.Entry

	bouquetclientset clientset.Interface

	shootsLister gardenv1beta1listers.ShootLister
	shootsSynced cache.InformerSynced

	addonManifestsLister listers.AddonManifestLister
	addonManifestsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

func NewController(
	log *logrus.Entry,
	bouquetclientset clientset.Interface,
	shootInformer gardenv1beta1informers.ShootInformer,
	addonManifestInformer v1alpha1informers.AddonManifestInformer,
) *Controller {

	bouquetscheme.AddToScheme(scheme.Scheme)
	controller := &Controller{
		log: log,

		bouquetclientset: bouquetclientset,

		shootsLister: shootInformer.Lister(),
		shootsSynced: shootInformer.Informer().HasSynced,

		addonManifestsLister: addonManifestInformer.Lister(),
		addonManifestsSynced: addonManifestInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Shoots"),
	}

	shootInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.shootAdd,
		UpdateFunc: controller.shootUpdate,
		DeleteFunc: controller.shootDelete,
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	c.log.Info("Starting Shoot-Bouquet compatibility controller")

	c.log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.shootsSynced, c.addonManifestsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	c.log.Infof("Starting %d workers", threadiness)
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.log.Info("Started workers")
	<-stopCh
	c.log.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)

		key, ok := obj.(string)
		if !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing %q: %s", key, err.Error())
		}

		c.workqueue.Forget(obj)
		c.log.Infof("Successfully synced %q", key)
		return nil
	}(obj)

	if err != nil {
		if c.workqueue.NumRequeues(obj) < maxRetries {
			c.log.Errorf("Error syncing key, retrying: %v", err)
			c.workqueue.AddRateLimited(obj)
		} else {
			runtime.HandleError(err)
		}
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	shoot, err := c.shootsLister.Shoots(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			c.log.Errorf("No last known state for deleted object %s/%s", namespace, name)
			return nil
		}

		return err
	}

	return c.reconcile(shoot)
}

func isTargetShoot(obj interface{}) bool {
	shoot, ok := obj.(*gardenv1beta1.Shoot)
	if !ok {
		return false
	}

	_, ok = shoot.Annotations[v1alpha1.AddonAnnotation]
	return ok
}

func (c *Controller) shootAdd(obj interface{}) {
	if !isTargetShoot(obj) {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

func (c *Controller) shootUpdate(oldObj, newObj interface{}) {
	if !isTargetShoot(newObj) {
		return
	}

	c.shootAdd(newObj)
}

func (c *Controller) shootDelete(obj interface{}) {
	if !isTargetShoot(obj) {
		return
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.log.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.workqueue.Add(key)
}
