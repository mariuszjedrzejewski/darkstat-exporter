package main

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	inBytesCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "darkstat_bytes_in_total",
			Help: "Incoming bytes",
		},
		[]string{"group", "ip", "hostname", "mac_address"})

	outBytesCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "darkstat_bytes_out_total",
			Help: "Outgoing bytes",
		},
		[]string{"group", "ip", "hostname", "mac_address"})

	totalBytesCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "darkstat_bytes_total",
			Help: "Total bytes",
		},
		[]string{"group", "ip", "hostname", "mac_address"})
)

func initPrometheusMetrics() {
	log.Println("registering prometheus metrics")

	prometheus.MustRegister(inBytesCounter)
	prometheus.MustRegister(outBytesCounter)
	prometheus.MustRegister(totalBytesCounter)
}
