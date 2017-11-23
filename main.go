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

func init() {
	pool = x509.NewCertPool()
	pool.AppendCertsFromPEM(pemCerts)
	netClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

func main() {

	loc, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	lastUpdated := time.Now().In(loc).Add(-24 * time.Hour)
	update(lastUpdated, loc)
	lastUpdated = time.Now()
	ticker := time.NewTicker(time.Minute * 5)
	for range ticker.C {
		update(lastUpdated, loc)
		lastUpdated = time.Now().In(loc)
	}
}

func update(lastUpdated time.Time, location *time.Location) {

	token := os.Getenv("GRABBER_TAG_TOKEN")
	if token == "" {
		fmt.Println("Requires env variable 'GRABBER_TAG_TOKEN'")
	}

	influxHost := os.Getenv("GRABBER_INFLUX_HOST")
	if influxHost == "" {
		fmt.Println("Requires env variable 'GRABBER_INFLUX_HOST'")
	}

	influxDB := os.Getenv("GRABBER_INFLUX_DB")
	if influxHost == "" {
		fmt.Println("Requires env variable 'GRABBER_INFLUX_DB'")
	}

	influxUser := os.Getenv("GRABBER_INFLUX_USER")
	if influxHost == "" {
		fmt.Println("Requires env variable 'GRABBER_INFLUX_USER'")
	}

	influxPassword := os.Getenv("GRABBER_INFLUX_PASSWORD")
	if influxHost == "" {
		fmt.Println("Requires env variable 'GRABBER_INFLUX_PASSWORD'")
	}

	if token == "" {
		os.Exit(1)
	}

	wirelessTags := wirelesstags.New(netClient, "https://www.mytaglist.com", token, location)

	tags, err := wirelessTags.Get(lastUpdated)
	if err != nil {
		log.Printf("Error on tag update: %v\n", err)
		return
	}

	fmt.Printf("Updated %d tags\n", len(tags))

	if influxHost == "" || influxDB == "" {
		os.Exit(1)
	}

	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     influxHost,
		Username: influxUser,
		Password: influxPassword,
	})

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  influxDB,
		Precision: "s",
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Create a point and add to batch
	for _, tag := range tags {
		for ts, metrics := range tag.Metrics {
			pt, err := influx.NewPoint("sensors", tag.Labels(), metrics, ts)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			bp.AddPoint(pt)
		}
	}

	// Write the batch
	if err := c.Write(bp); err != nil {
		//log.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d metric points\n", len(bp.Points()))
}
