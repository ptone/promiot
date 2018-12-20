package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ptone/promiot"
)

// to run, set the environment variables like:
// GCLOUD_PROJECT=ptone-serverless METRICS_SUBSCRIPTION=metric-pull go run main.go
func main() {
	project := os.Getenv("GCLOUD_PROJECT")
	if project == "" {
		log.Fatal("GCLOUD_PROJECT env var must be set")
	}
	subscription := os.Getenv("METRICS_SUBSCRIPTION")
	if subscription == "" {
		log.Fatal("METRICS_SUBSCRIPTION env var must be set")
	}
	// A receiver receives encoded metrics on PubSub topic, and unpacks them into a for ready to be gathered on scrape
	receiver := promiot.NewPromiotReceiver(project, subscription)

	// The receiver is added to the array of default gatherers
	prometheus.DefaultGatherer = prometheus.Gatherers{
		prometheus.DefaultGatherer,
		receiver,
	}

	// start pulling messages from the pubsub topic
	go receiver.Receive()

	// expose the collected metrics to be scraped by prometheus
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
