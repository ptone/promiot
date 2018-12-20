package promiot

import (
	"context"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"cloud.google.com/go/pubsub"
)

// PromiotReceiver combines pubsub topic with a gatherable cache of received metrics
// implements the prometheus.Gatherer interface
type PromiotReceiver struct {
	received            map[string]*MetricBundle
	client              *pubsub.Client
	sub                 *pubsub.Subscription
	ctx                 context.Context
	recLatencyHistogram *prometheus.HistogramVec
}

// NewPromiotReceiver - constructor
func NewPromiotReceiver(project string, subscription string) *PromiotReceiver {
	r := &PromiotReceiver{}
	r.ctx = context.Background()
	var err error
	r.client, err = pubsub.NewClient(r.ctx, project)
	if err != nil {
		log.Fatalf("Could not create pubsub Client: %v", err)
	}
	r.received = make(map[string]*MetricBundle)
	r.sub = r.client.Subscription(subscription)
	r.recLatencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "promiot_receive_latency_seconds",
		Help:    "MQTT + Pubsub latency distributions.",
		Buckets: prometheus.LinearBuckets(0.05, 0.05, 20),
	},
		// TODO will continue to investigate ways to convey the device location as a label
		// []string{"device", "location"})

		[]string{"instance"})
	prometheus.MustRegister(r.recLatencyHistogram)
	return r
}

// Receive - uses cloud pubsub streaming pull to receive metrics over telemetry and decode
func (r *PromiotReceiver) Receive() {
	err := r.sub.Receive(r.ctx, func(ctx context.Context, msg *pubsub.Message) {
		log.Println("message rec")
		bundle := &MetricBundle{}
		err := proto.Unmarshal(msg.Data, bundle)
		if err != nil {

			log.Println("unmarshaling error: ", err)
			msg.Ack()
			return

		}
		// fmt.Println(bundle.BundleTimestamp)
		sendTime := time.Unix(0, bundle.BundleTimestamp)
		recLatency := time.Since(sendTime).Seconds()
		r.recLatencyHistogram.WithLabelValues(msg.Attributes["deviceId"]).Observe(recLatency)
		// r.recLatencyHistogram.WithLabelValues("foo", "bar").Observe(recLatency)
		if len(bundle.Families) > 0 {
			r.received[msg.Attributes["deviceNumId"]] = bundle
			// fmt.Printf("%v\n", bundle.Families[0])
		}
		msg.Ack()
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Gather : Implement the prometheus "Gather" interface
func (r *PromiotReceiver) Gather() ([]*dto.MetricFamily, error) {
	var allFamilies []*dto.MetricFamily
	for _, v := range r.received {
		allFamilies = append(allFamilies, v.Families...)
	}
	// TODO consider reset after gather? At this point stale metrics will linger in the received data structure
	return allFamilies, nil
}
