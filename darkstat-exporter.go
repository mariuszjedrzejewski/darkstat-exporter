package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	"github.com/go-co-op/gocron/v2"
)

type HostData struct {
	Hostname   string
	MacAddress string
	In         float64
	Out        float64
	Total      float64
}

var (
	v         *viper.Viper
	scheduler gocron.Scheduler
	updateJob gocron.Job
)

func init() {
	initViperConfig()

	initPrometheusMetrics()

	log.Println("getting initial metrics values")
	recordMetrics()

	var err error
	scheduler, err = gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		log.Fatal(err)
	}

	metricsRecordInterval := v.GetDuration("metricsRecordInterval")
	log.Printf("starting scheduler job for every %s", metricsRecordInterval)
	updateJob, err = scheduler.NewJob(gocron.DurationJob(metricsRecordInterval), gocron.NewTask(recordMetrics))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("registered job: %s, id: %s", updateJob.Name(), updateJob.ID())

	scheduler.Start()
}

func recordMetrics() {
	log.Println("running recordMetrics()")

	groups := v.GetStringMapStringSlice("groups")
	for group, ips := range groups {
		log.Printf("group: %s", group)
		for _, ip := range ips {
			url := fmt.Sprintf(v.GetString("darkstatUrlPrefix"), ip)

			log.Printf("getting data for url: %s", url)
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

			inBytesCounter.WithLabelValues(group, ip, d.Hostname, d.MacAddress).Set(float64(d.In))
			outBytesCounter.WithLabelValues(group, ip, d.Hostname, d.MacAddress).Set(float64(d.Out))
			totalBytesCounter.WithLabelValues(group, ip, d.Hostname, d.MacAddress).Set(float64(d.Total))
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
		log.Printf("error converting value")
		return 0
	}
	return rawValue
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	listenPort := v.GetString("listenPort")
	log.Printf("starting server at %s", listenPort)
	log.Fatal(http.ListenAndServe(listenPort, nil))
}
