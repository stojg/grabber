package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/stojg/grabber/lib/wirelesstags"
)

var (
	netClient *http.Client
	pool      *x509.CertPool
)

const (
	MyDB     = "mydb"
	username = "bubba"
	password = "bumblebeetuna"
)

func init() {
	pool = x509.NewCertPool()
	pool.AppendCertsFromPEM(pemCerts)
	netClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

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
		fmt.Println("Requires env variable 'GRABBER_WIRELESSTAG_TOKEN'")
	}

	influxdbHost := os.Getenv("GRABBER_INFLUX_URL")
	if influxdbHost == "" {
		fmt.Println("Requires env variable 'GRABBER_INFLUX_URL'")
	}

	if token == "" || influxdbHost == "" {
		os.Exit(1)
	}

	wirelessTags := wirelesstags.New(netClient, "https://www.mytaglist.com", token)

	tags, err := wirelessTags.Get(lastUpdated)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Updated %d tags\n", len(tags))

	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     influxdbHost,
		Username: username,
		Password: password,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
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
			pt, err := influx.NewPoint("sensors", metricTags, metrics, ts)
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
