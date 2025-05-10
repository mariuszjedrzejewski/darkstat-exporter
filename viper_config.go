package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
)

func initViperConfig() {
	v = viper.New()
	v.SetDefault("metricsRecordInterval", time.Duration(30*time.Second))
	v.SetDefault("listenPort", ":80")

	v.AddConfigPath(".")
	v.AddConfigPath("/")
	v.SetConfigFile("config.yaml")

	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	log.Printf("config file used: %s", v.ConfigFileUsed())

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)

		// re-read config
		v.ReadInConfig()

		// update cron job with current duration (as may change in config update)
		scheduler.Update(updateJob.ID(), gocron.DurationJob(v.GetDuration("metricsRecordInterval")), gocron.NewTask(recordMetrics))
	})
	v.WatchConfig()
}
