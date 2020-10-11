package main

import (
	"encoding/json"
	"flag"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

type ChangeEvent struct {
	Event        string  `json:"event"`
	Version      int64   `json:"version"`
	Availability float64 `json:"availability"` // this is the result of Status.DesiredReplicas / Spec.Replicas
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.WithField("component", "notifier")
	events := make(chan *ChangeEvent)
	hub := NewHub()
	selector := flag.String("l", "", "Specify a kubernetes label selector")
	flag.Parse()

	if *selector == "" {
		log.Errorf("no selector specified")
		os.Exit(1)
	}

	config, _ := rest.InClusterConfig()
	client, _ := kubernetes.NewForConfig(config)

	log.WithField("label", *selector).Debugf("starting watcher for label")
	watcher := &DeploymentWatcher{Events: events, Client: client, LabelSelector: *selector}

	go hub.Run()
	go watcher.Run()

	go func() {
		for {
			data, err := json.Marshal(<-events)
			if err != nil {
				log.Error("failed to marshal event to JSON")
				continue
			}
			hub.Broadcast(data)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})
	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
