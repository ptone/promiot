package promiot

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
)

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func NewRegistry() (prometheus.Registerer, prometheus.Gatherer, error) {
	var (
		r                              = prometheus.NewRegistry()
		registry prometheus.Registerer = r
		gatherer prometheus.Gatherer   = r
	)
	return registry, gatherer, nil
}

type Promiot struct {
	registry              prometheus.Registerer
	Gatherer              prometheus.Gatherer
	client                mqtt.Client
	topic                 string
	acktimer              map[uint16]int64
	ackDurationsHistogram *prometheus.HistogramVec
	defaultLabels         prometheus.Labels
	startTime             time.Time
	startGauge            *prometheus.GaugeVec
	PublishDelay          int64
}

func NewPromiot(client mqtt.Client, topic string, defaultLabels map[string]string) (p *Promiot, err error) {
	p = &Promiot{}
	p.startTime = time.Now()
	p.registry, p.Gatherer, err = NewRegistry()
	if err != nil {
		return nil, err
	}
	p.client = client
	p.topic = topic
	// p.defaultLabels = make(map[string]string)
	p.defaultLabels = defaultLabels
	labelKeys := make([]string, 0, len(p.defaultLabels))
	for k := range p.defaultLabels {
		labelKeys = append(labelKeys, k)
	}
	p.acktimer = make(map[uint16]int64)
	p.ackDurationsHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mqtt_telemetry_puback_latency",
		Help:    "MQTT ack latency distributions.",
		Buckets: prometheus.LinearBuckets(50, 20, 20),
	}, labelKeys)
	p.startGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "promiot_relay_start_unix",
		Help: "help todo",
	}, labelKeys)
	p.registry.MustRegister(p.ackDurationsHistogram, p.startGauge)
	p.startGauge.With(defaultLabels).Set(float64(p.startTime.Unix()))
	p.PublishDelay = 60
	return p, nil
}

func (p *Promiot) Publish() (err error) {
	mfs, err := p.Gatherer.Gather()
	if len(mfs) == 0 {
		log.Println("no metrics")
		// TODO if there are no metrics nothing gets published
		// need at least one metric other than self metric latency histo
		// return nil
	}
	log.Println("gathered")
	if err != nil {
		return err
	}
	bundle := &MetricBundle{Families: mfs, BundleTimestamp: time.Now().UnixNano()}
	fmt.Println(bundle.BundleTimestamp)
	// fmt.Printf("%v\n", bundle.Families[0])

	data, err := proto.Marshal(bundle)
	if err != nil {
		//log.Fatal("marshaling error: ", err)
		return err
	}
	token := p.client.Publish(
		p.topic,
		// "/foo/invalid",
		1,
		false,
		data).(*mqtt.PublishToken)
	p.acktimer[token.MessageID()] = makeTimestamp()
	token.WaitTimeout(5 * time.Second)

	v := float64(makeTimestamp() - p.acktimer[token.MessageID()])
	// TODO - determine if any labels needed here or not
	p.ackDurationsHistogram.With(p.defaultLabels).Observe(v)
	return nil
}

// functions:
// register
// publish
// schedule, repeating
