package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/stojg/grabber/lib/wirelesstags"
)

var (
	netClient *http.Client
)

const (
	// how many points are we going to send to influxDB in one request
	influxMaxBatch = 1000
)

func init() {
	netClient = &http.Client{Transport: &http.Transport{}}
}

func main() {

	influxHost := os.Getenv("GRABBER_INFLUX_HOST")
	if influxHost == "" {
		handleError(errors.New("requires ENV variable 'GRABBER_INFLUX_HOST'"))
	}

	influxUser := os.Getenv("GRABBER_INFLUX_USER")
	if influxHost == "" {
		handleError(errors.New("requires ENV variable 'GRABBER_INFLUX_USER'"))
	}

	influxPassword := os.Getenv("GRABBER_INFLUX_PASSWORD")
	if influxHost == "" {
		handleError(errors.New("requires ENV variable 'GRABBER_INFLUX_PASSWORD'"))
	}

	influxDB := os.Getenv("GRABBER_INFLUX_DB")
	if influxHost == "" {
		handleError(errors.New("requires ENV variable 'GRABBER_INFLUX_DB'"))
	}

	wirelessTagToken := os.Getenv("GRABBER_TAG_TOKEN")
	if wirelessTagToken == "" {
		handleError(errors.New("requires ENV variable 'GRABBER_TAG_TOKEN'"))
	}

	influxClient, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:      influxHost,
		Username:  influxUser,
		Password:  influxPassword,
		UserAgent: "grabber",
	})
	handleError(err)

	// the timezone should be set to what the wireless tags have been set to
	loc, err := time.LoadLocation("Pacific/Auckland")
	handleError(err)

	tagClient, err := wirelesstags.NewHTTPClient(wirelesstags.HTTPConfig{
		Addr:     "https://www.mytaglist.com",
		Token:    wirelessTagToken,
		Location: loc,
	})
	handleError(err)

	lastUpdated := time.Now().In(loc).Add(-24 * time.Hour)
	for ticker := time.NewTicker(time.Minute * 5); true; <-ticker.C {
		fmt.Fprintln(os.Stdout, "starting update")
		if err := update(tagClient, influxClient, influxDB, lastUpdated); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		lastUpdated = time.Now().In(loc)
		fmt.Fprintln(os.Stdout, "update done")
	}
}

func update(wirelessTags *wirelesstags.Client, influxClient influx.Client, databaseName string, from time.Time) error {

	tags, err := wirelessTags.Get(from)
	if err != nil {
		return fmt.Errorf("on tag update: %v", err)
	}

	bp := getNewPointBatch(databaseName)

	for _, tag := range tags {
		for unixTime, metrics := range tag.Metrics {
			wrote, err := addPoint(influxClient, bp, tag.Labels(), metrics, unixTime)
			if err != nil {
				return err
			}
			if wrote {
				bp = getNewPointBatch(databaseName)
			}
		}
	}
	return writePoints(influxClient, bp)
}

func addPoint(c influx.Client, bp influx.BatchPoints, tags map[string]string, metrics []*wirelesstags.Metric, unix int64) (bool, error) {
	data := make(map[string]interface{})

	for _, m := range metrics {
		data[m.Name()] = m.Value()
	}

	pt, err := influx.NewPoint("sensors", tags, data, time.Unix(unix, 0))
	if err != nil {
		return false, fmt.Errorf("error: %v", err)
	}
	bp.AddPoint(pt)

	if len(bp.Points()) >= influxMaxBatch {
		if err := writePoints(c, bp); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func writePoints(c influx.Client, bp influx.BatchPoints) error {
	if len(bp.Points()) == 0 {
		return nil
	}

	if err := c.Write(bp); err != nil {
		return fmt.Errorf("database write error: %v", err)
	}

	return nil
}

func getNewPointBatch(influxDB string) influx.BatchPoints {

	// Create a new point batch
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  influxDB,
		Precision: "s",
	})
	// it will only fail it it can't parse the duration
	if err != nil {
		panic(err)
	}
	return bp
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
