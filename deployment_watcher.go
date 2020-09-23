package main

import (
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

type DeploymentWatcher struct {
	Client        *kubernetes.Clientset
	LabelSelector string
	Events        chan *ChangeEvent
}

func (w *DeploymentWatcher) Run() {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(w.Client, time.Second, informers.WithTweakListOptions(func(options *v12.ListOptions) {
		options.LabelSelector = w.LabelSelector // set a custom label selector so we're only dealing with one resource
	}))
	deploymentInformer := informerFactory.Apps().V1().Deployments().Informer()

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			w.Events <- &ChangeEvent{
				Event:        "created",
				Version:      deployment.Status.ObservedGeneration,
				Availability: float64(deployment.Status.UpdatedReplicas) / float64(*deployment.Spec.Replicas),
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			deployment := newObj.(*v1.Deployment)
			w.Events <- &ChangeEvent{
				Event:        "updated",
				Version:      deployment.Status.ObservedGeneration,
				Availability: float64(deployment.Status.UpdatedReplicas) / float64(*deployment.Spec.Replicas),
			}
		},
		DeleteFunc: func(obj interface{}) {
			w.Events <- &ChangeEvent{
				Event: "deleted",
			}
		},
	})
}
