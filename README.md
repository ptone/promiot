# Promiot

Promiot is an experimental library to allow devices to use [Prometheus](https://prometheus.io/) client [library](https://prometheus.io/docs/instrumenting/clientlibs/) instrumentation locally on device, and convey the metrics over telemetry via [Google Cloud IoT Core](https://cloud.google.com/iot/docs/) to be scraped by a Cloud hosted instance of Prometheus for IoT Remote Monitoring.

In golang this can be achieved by directly integrating promiot into your code.

For other languages, a side-car process scrapes metrics locally before sending over MQTT.

See examples folder for demonstration of these patterns.

Promiot provides a built in metric to track the latency of mqtt ack latency:
`promiot_mqtt_telemetry_puback_latency`

As well as latency of publishing to the cloud: `promiot_receive_latency_seconds`

This is not an official Google Product.