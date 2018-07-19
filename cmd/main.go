package main

import (
	"flag"
	bouquet "github.com/gardener/bouquet/pkg/client/clientset/versioned"
	"github.com/gardener/bouquet/pkg/client/informers/externalversions"
	"github.com/gardener/bouquet/pkg/controller/instance"
	"github.com/gardener/bouquet/pkg/controller/shoot"
	"github.com/gardener/bouquet/pkg/signals"
	garden "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	gardenexternalversions "github.com/gardener/gardener/pkg/client/garden/informers/externalversions"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
	"time"
)

var (
	masterURL  string
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

type controller interface {
	Run(threadiness int, stopChan <-chan struct{}) error
}

// TODO: Split up main in testable components
func main() {
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	bouquetClient, err := bouquet.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building bouquet clientset: %s", err.Error())
	}

	gardenClient, err := garden.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building garden clientset: %s", err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building dynamic clientset: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(kubeClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			restMapper.Reset()
		}
	}()

	bouquetInformerFactory := externalversions.NewSharedInformerFactory(bouquetClient, 30*time.Second)
	gardenInformerFactory := gardenexternalversions.NewSharedInformerFactory(gardenClient, 30*time.Second)

	instanceController := instance.NewController(
		logrus.NewEntry(logrus.StandardLogger()),
		kubeClient,
		bouquetClient,
		gardenClient,
		dynamicClient,
		restMapper,
		bouquetInformerFactory.Garden().V1alpha1().AddonInstances(),
		bouquetInformerFactory.Garden().V1alpha1().AddonManifests(),
	)

	shootController := shoot.NewController(
		logrus.NewEntry(logrus.StandardLogger()),
		bouquetClient,
		gardenInformerFactory.Garden().V1beta1().Shoots(),
		bouquetInformerFactory.Garden().V1alpha1().AddonManifests(),
	)

	go bouquetInformerFactory.Start(stopCh)
	go gardenInformerFactory.Start(stopCh)

	// TODO: Implement proper health service depending on state of controllers
	controllers := []controller{instanceController, shootController}
	var wg sync.WaitGroup
	wg.Add(len(controllers))
	for _, c := range controllers {
		go func(c controller) {
			defer wg.Done()
			if err := c.Run(2, stopCh); err != nil {
				logrus.Errorf("Error running controller: %s", err.Error())
			}
		}(c)
	}

	wg.Wait()
}
