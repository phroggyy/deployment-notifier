package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
)

type ChangeEvent struct {
	Event        string  `json:"event"`
	Version      int64   `json:"version"`
	Availability float64 `json:"availability"` // this is the result of Status.DesiredReplicas / Spec.Replicas
}

func main() {
	events := make(chan *ChangeEvent)
	hub := NewHub()

	config, _ := clientcmd.BuildConfigFromFlags("", "")
	client, _ := kubernetes.NewForConfig(config)

	watcher := &DeploymentWatcher{Events: events, Client: client}

	go hub.Run()
	go watcher.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})
	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
