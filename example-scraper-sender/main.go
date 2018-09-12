package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ptone/promiot"

	dto "github.com/prometheus/client_model/go"
)

var (
	scrapeURL = "http://localhost:9090/metrics"
	deviceID  = flag.String("device", "", "Cloud IoT Core Device ID")
	bridge    = struct {
		host *string
		port *string
	}{
		flag.String("mqtt_host", "mqtt.googleapis.com", "MQTT Bridge Host"),
		flag.String("mqtt_port", "443", "MQTT Bridge Port"),
	}
	projectID  = flag.String("project", "", "GCP Project ID")
	registryID = flag.String("registry", "", "Cloud IoT Registry ID (short form)")
	region     = flag.String("region", "", "GCP Region")
	certsCA    = flag.String("ca_certs", "", "Download https://pki.google.com/roots.pem")
	privateKey = flag.String("private_key", "", "Path to private key file")
)

func main() {
	flag.Parse()

	log.Println("[main] Loading Google's roots")
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(*certsCA)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	log.Println("[main] Creating TLS Config")

	config := &tls.Config{
		RootCAs:            certpool,
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{},
		MinVersion:         tls.VersionTLS12,
	}

	clientID := fmt.Sprintf("projects/%v/locations/%v/registries/%v/devices/%v",
		*projectID,
		*region,
		*registryID,
		*deviceID,
	)

	log.Println("[main] Creating MQTT Client Options")
	opts := MQTT.NewClientOptions()

	broker := fmt.Sprintf("ssl://%v:%v", *bridge.host, *bridge.port)
	log.Printf("[main] Broker '%v'", broker)

	opts.AddBroker(broker)
	opts.SetClientID(clientID).SetTLSConfig(config)

	opts.SetUsername("unused")

	// TODO set up token refresh
	token := jwt.New(jwt.SigningMethodES256)
	token.Claims = jwt.StandardClaims{
		Audience:  *projectID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	log.Println("[main] Load Private Key")
	keyBytes, err := ioutil.ReadFile(*privateKey)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[main] Parse Private Key")
	// key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	key, err := jwt.ParseECPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[main] Sign String")
	tokenString, err := token.SignedString(key)
	if err != nil {
		log.Fatal(err)
	}

	opts.SetPassword(tokenString)

	opts.KeepAlive = 10
	log.Println("[main] MQTT Client Connecting")
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	topic := struct {
		config    string
		telemetry string
	}{
		config:    fmt.Sprintf("/devices/%v/config", *deviceID),
		telemetry: fmt.Sprintf("/devices/%v/events", *deviceID),
	}

	labels := map[string]string{
		// https://en.wikipedia.org/wiki/ISO_3166-2:US
		"location": "US-CA",
		// Replace with a device ID
		"instance": deviceID,
	}
	p, _ := promiot.NewPromiot(client, fmt.Sprintf("%s/metrics", topic.telemetry), labels)

	p.Gatherer = prometheus.Gatherers{
		p.Gatherer,
		prometheus.GathererFunc(func() ([]*dto.MetricFamily, error) { return promiot.FetchMetricFamilies(scrapeURL) }),
	}

	for {
		log.Printf("[main] Publishing Message #%d", 0)
		p.Publish()
		time.Sleep(time.Second * p.PublishDelay)
	}

	log.Println("[main] MQTT Client Disconnecting")
	client.Disconnect(250)
	log.Println("[main] Done")
}
