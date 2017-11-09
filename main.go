package main

import (
	"fmt"
	"log"
    "os"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/stojg/grabber/lib/wirelesstags"

)

const (
	MyDB     = "mydb"
	username = "bubba"
	password = "bumblebeetuna"
)

func main() {
	lastUpdated := time.Now().Add(-24 * time.Hour)
	update(lastUpdated)
	lastUpdated = time.Now()
	ticker := time.NewTicker(time.Minute * 5)
	for range ticker.C {
		update(lastUpdated)
		lastUpdated = time.Now()
	}

}

func update(lastUpdated time.Time) {

    token := os.Getenv("GRABBER_WIRELESSTAG_TOKEN")
    if token == "" {
        fmt.Println("Need GRABBER_WIRELESSTAG_TOKEN")
        os.Exit(1)
    }

	influxdbHost := os.Getenv("GRABBER_INFLUX_URL")
	if influxdbHost == "" {
		fmt.Println("Need GRABBER_INFLUX_URL")
		os.Exit(1)
	}

	tags, err := wirelesstags.Get(token, "https://www.mytaglist.com", lastUpdated)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Updated %d tags\n", len(tags))

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://localhost:8086",
		Username: username,
		Password: password,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  MyDB,
		Precision: "s",
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Create a point and add to batch
	for _, tag := range tags {
		metricTags := map[string]string{
			"tag":      tag.Name,
			"location": "16moir",
		}

		for ts, metrics := range tag.Metrics {
			pt, err := client.NewPoint("sensors", metricTags, metrics, ts)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			bp.AddPoint(pt)
		}
	}

	// Write the batch
	if err := c.Write(bp); err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d metric points\n", len(bp.Points()))
}
