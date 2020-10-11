package main

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type DeploymentWatcher struct {
	Client        *kubernetes.Clientset
	LabelSelector string
	Events        chan *ChangeEvent
}

const revisionAnnotation = "deployment.kubernetes.io/revision"

func (w *DeploymentWatcher) Run() {
	var currentRevision atomic.Value

	log := logrus.WithField("component", "deployment-watcher")
	informerFactory := informers.NewSharedInformerFactoryWithOptions(w.Client, time.Second, informers.WithTweakListOptions(func(options *v12.ListOptions) {
		options.LabelSelector = w.LabelSelector // set a custom label selector so we're only dealing with one resource
	}))
	deploymentInformer := informerFactory.Apps().V1().Deployments().Informer()

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			revision := deployment.Annotations[revisionAnnotation]
			currentRevision.Store(revision)

			//deployment.GetResourceVersion()
			log.Debug("RS created")
			w.Events <- &ChangeEvent{
				Event:        "created",
				Version:      deployment.Status.ObservedGeneration,
				Availability: float64(deployment.Status.AvailableReplicas) / float64(*deployment.Spec.Replicas),
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment := oldObj.(*v1.Deployment)
			deployment := newObj.(*v1.Deployment)

			// if we haven't seen changes for relevant fields, we won't emit an event
			if oldDeployment.Generation == deployment.Generation &&
				oldDeployment.Status.AvailableReplicas == deployment.Status.AvailableReplicas &&
				oldDeployment.Spec.Replicas == deployment.Spec.Replicas {
				return
			}

			revision := deployment.Annotations[revisionAnnotation]
			currentRevision.Store(revision)

			//w.Events <- &ChangeEvent{
			//	Event:        "updated",
			//	Version:      deployment.Generation,
			//	Availability: float64(deployment.Status.AvailableReplicas) / float64(*deployment.Spec.Replicas),
			//}
		},
		DeleteFunc: func(obj interface{}) {
			log.Debug("deployment deleted")
			w.Events <- &ChangeEvent{
				Event: "deleted",
			}
		},
	})

	rsInformer := informerFactory.Apps().V1().ReplicaSets().Informer()
	rsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rs := obj.(*v1.ReplicaSet)
			revision := currentRevision.Load()
			if revision != rs.Annotations[revisionAnnotation] {
				return
			}

			version, _ := strconv.Atoi(revision.(string))
			w.Events <- &ChangeEvent{
				Event:        "updated",
				Version:      int64(version),
				Availability: float64(rs.Status.Replicas) / float64(rs.Status.ReadyReplicas),
			}

			log.
				WithField("revision", revision).
				WithField("pod-template-hash", rs.Labels["pod-template-hash"]).
				Debug("new replica set added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			rs := newObj.(*v1.ReplicaSet)
			oldRs := oldObj.(*v1.ReplicaSet)
			revision := currentRevision.Load()
			if revision != rs.Annotations[revisionAnnotation] {
				return
			}

			if oldRs.Generation == rs.Generation &&
				oldRs.Status.ReadyReplicas == rs.Status.ReadyReplicas &&
				oldRs.Spec.Replicas == rs.Spec.Replicas {
				return
			}

			version, _ := strconv.Atoi(revision.(string))
			w.Events <- &ChangeEvent{
				Event:        "updated",
				Version:      int64(version),
				Availability: float64(rs.Status.ReadyReplicas) / float64(*rs.Spec.Replicas),
			}
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	log.Debug("started informer")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		deploymentInformer.Run(stop)
		wg.Done()
	}()
	go func() {
		rsInformer.Run(stop)
		wg.Done()
	}()

	wg.Wait()
}
