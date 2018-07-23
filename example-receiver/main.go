package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ptone/promiot"
)

func main() {
	project := os.Getenv("GCLOUD_PROJECT")
	if project == "" {
		log.Fatal("GCLOUD_PROJECT env var must be set")
	}
	subscription := os.Getenv("METRICS_SUBSCRIPTION")
	if subscription == "" {
		log.Fatal("METRICS_SUBSCRIPTION env var must be set")
	}
	receiver := promiot.NewPromiotReceiver(project, subscription)
	prometheus.DefaultGatherer = prometheus.Gatherers{
		prometheus.DefaultGatherer,
		receiver,
	}
	go receiver.Receive()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
