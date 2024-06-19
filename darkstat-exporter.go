package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-co-op/gocron"
)

type HostData struct {
	Hostname   string
	MacAddress string
	In         float64
	Out        float64
	Total      float64
}

type ConfigEntry struct {
	Group string   `json:"group"`
	Ip    []string `json:"ip"`
}

type Config struct {
	Cfg []ConfigEntry `json:"cfg"`
}

var config Config

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

func init() {
	err := json.Unmarshal([]byte(os.Getenv("CONFIG")), &config)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Registering prometheus metrics...")
	prometheus.MustRegister(inBytesCounter)
	prometheus.MustRegister(outBytesCounter)
	prometheus.MustRegister(totalBytesCounter)

	log.Println("Getting initial metrics values...")
	recordMetrics()

	metricsRecordInterval := 30
	if len(os.Getenv("METRICS_RECORD_INTERVAL")) != 0 {
		interval, err := strconv.ParseInt(os.Getenv("METRICS_RECORD_INTERVAL"), 10, 64)
		if err == nil {
			metricsRecordInterval = int(interval)
		}
	}
	log.Printf("Starting scheduler for every %d sec...", metricsRecordInterval)
	s := gocron.NewScheduler(time.UTC)
	s.Every(metricsRecordInterval).Seconds().Do(recordMetrics)
	s.StartAsync()
}

var recordMetrics = func() {
	for _, cfg := range config.Cfg {
		for _, ip := range cfg.Ip {
			url := fmt.Sprintf(os.Getenv("DARKSTAT_URL_PREFIX"), ip)

			log.Printf("Getting data for url: %s", url)
			response, err := http.Get(url)
			if err != nil {
				log.Println(err)
				continue
			}

			dataToParse := make([]string, 0)
			doc, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				log.Println(err)
				continue
			}

			doc.Find("p").Each(func(x int, p *goquery.Selection) {
				if x == 0 || x == 2 {
					dataToParse = append(dataToParse, strings.Split(p.Text(), "\n")...)
				}
			})

			d := getValues(dataToParse)

			inBytesCounter.WithLabelValues(cfg.Group, ip, d.Hostname, d.MacAddress).Set(float64(d.In))
			outBytesCounter.WithLabelValues(cfg.Group, ip, d.Hostname, d.MacAddress).Set(float64(d.Out))
			totalBytesCounter.WithLabelValues(cfg.Group, ip, d.Hostname, d.MacAddress).Set(float64(d.Total))
		}
	}
}

func getValues(data []string) HostData {
	hd := HostData{}

	for _, line := range data {
		if len(line) == 0 {
			continue
		}

		trimmedLine := strings.TrimLeft(line, " ")
		lineData := strings.Split(trimmedLine, ": ")
		rawValueStr := strings.ReplaceAll(lineData[1], ",", "")

		switch lineData[0] {
		case "Hostname":
			hd.Hostname = rawValueStr
		case "MAC Address":
			hd.MacAddress = rawValueStr
		case "In":
			hd.In = getRawValue(rawValueStr)
		case "Out":
			hd.Out = getRawValue(rawValueStr)
		case "Total":
			hd.Total = getRawValue(rawValueStr)
		}
	}

	return hd
}

func getRawValue(s string) float64 {
	rawValue, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Error converting value")
		return 0
	}
	return rawValue
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	port := ":9090"
	if len(os.Getenv("LISTEN_PORT")) != 0 {
		port = os.Getenv("LISTEN_PORT")
	}
	log.Printf("Starting server at %s...", port)
	http.ListenAndServe(port, nil)
}
