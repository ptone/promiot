package promiot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"cloud.google.com/go/pubsub"
)

type PromiotReceiver struct {
	// registry TODO for metadata
	received            map[string]*MetricBundle
	client              *pubsub.Client
	sub                 *pubsub.Subscription
	ctx                 context.Context
	recLatencyHistogram *prometheus.HistogramVec
}

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
		[]string{"device", "location"})
	prometheus.MustRegister(r.recLatencyHistogram)
	return r
}

func (r *PromiotReceiver) Receive() {
	err := r.sub.Receive(r.ctx, func(ctx context.Context, msg *pubsub.Message) {
		log.Println("message rec")
		bundle := &MetricBundle{}
		err := proto.Unmarshal(msg.Data, bundle)
		if err != nil {
			log.Fatal("unmarshaling error: ", err)
		}
		// fmt.Println(bundle.BundleTimestamp)
		sendTime := time.Unix(0, bundle.BundleTimestamp)
		recLatency := time.Since(sendTime).Seconds()
		fmt.Println("overall")
		fmt.Println(sendTime)
		fmt.Println(recLatency)
		fmt.Println("publish")
		fmt.Println(msg.PublishTime)
		fmt.Println(time.Since(msg.PublishTime))
		r.recLatencyHistogram.WithLabelValues("foo", "bar").Observe(recLatency)
		fmt.Println(len(msg.Data))
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

func (r *PromiotReceiver) Gather() ([]*dto.MetricFamily, error) {
	var allFamilies []*dto.MetricFamily
	for _, v := range r.received {
		// TODO consider injecting label per sender
		// this would increase cardinality - but preserve sums across otherwise similar
		// metrics such as counters or distributions which were not instrumented as vectors
		// with labels from the source.
		// for point source cardinality - this is easy - the source should basically
		// ship the "instance" tag explicitly, it is harder for intermediate cardinality
		// such as latency by region. The proper solution is likely to enforce high
		// cardinality, then federate an aggreation to a user-facing prometheus using
		// prometheus federation feature
		allFamilies = append(allFamilies, v.Families...)
	}
	// TODO reset after gather?
	return allFamilies, nil
}
