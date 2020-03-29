package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/stojg/grabber/internal/config"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/stojg/grabber/lib/wirelesstags"
)

const (
	// how many points are we going to send to influxDB in one request
	influxMaxBatch = 1000
)

var debugMode = false

func main() {

	cfg, err := config.Load()
	handleError(err)

	influxClient, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:      cfg.InfluxDB.Host,
		Username:  cfg.InfluxDB.User,
		Password:  cfg.InfluxDB.Password,
		UserAgent: "grabber",
	})
	handleError(err)

	// the timezone should be set to what the wireless tags have been set to
	loc, err := time.LoadLocation("Pacific/Auckland")
	handleError(err)

	tagClient, err := wirelesstags.NewHTTPClient(wirelesstags.HTTPConfig{
		Addr:     "https://www.mytaglist.com",
		Token:    cfg.WirelessTag.Token,
		Location: loc,
	})
	handleError(err)

	updateTick := time.Minute * 5
	lastUpdated := time.Now().In(loc).Add(-90 * 24 * time.Hour)
	for ticker := time.NewTicker(updateTick); true; <-ticker.C {
		if err := update(tagClient, influxClient, cfg.InfluxDB.DB, lastUpdated); err != nil {
			logf("Update failure: %s\n", err)
			logf("Will retry the update in %s\n", updateTick)
		}
		lastUpdated = time.Now().In(loc)
	}
}

func debug(a string) {
	if !debugMode {
		return
	}
	fmt.Println(a)
}

func debugf(format string, a ...interface{}) {
	if !debugMode {
		return
	}
	fmt.Printf(format, a...)
}

func update(wirelessTags *wirelesstags.Client, influxClient influx.Client, databaseName string, fromTime time.Time) error {

	debugf("fetching wireless tag data from %s\n", fromTime)
	tags, err := wirelessTags.Get(fromTime)
	if err != nil {
		return fmt.Errorf("wirelessTags.Get - %v", err)
	}

	bp := getNewPointBatch(databaseName)

	debugf("data from %d sensors found\n", len(tags))
	for _, tag := range tags {
		for unixTime, metrics := range tag.Metrics {
			err := addPoint(bp, tag.Labels(), metrics, unixTime)
			if err != nil {
				return err
			}

			if len(bp.Points()) >= influxMaxBatch {
				if err := writePoints(influxClient, bp, 1); err != nil {
					return err
				}
				bp = getNewPointBatch(databaseName)
			}
		}
	}
	// flush out the last points
	return writePoints(influxClient, bp, 1)
}

func addPoint(bp influx.BatchPoints, tags map[string]string, metrics []*wirelesstags.Metric, unix int64) error {
	data := make(map[string]interface{})
	for _, m := range metrics {
		data[m.Name()] = m.Value()
	}
	pt, err := influx.NewPoint("sensors", tags, data, time.Unix(unix, 0))
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	bp.AddPoint(pt)
	return nil
}

func writePoints(c influx.Client, bp influx.BatchPoints, attempt int) error {
	if len(bp.Points()) == 0 {
		debugf("no data points to write into influxdb\n")
		return nil
	}

	debugf("writing %d points into influxdb\n", len(bp.Points()))
	err := c.Write(bp)

	if err == nil {
		debugf("%d points were written into influxdb\n", len(bp.Points()))
		return nil
	}

	if attempt >= 15 {
		return fmt.Errorf("writing to influxdb failed after %d attempts: %s", attempt, err)
	}

	base := 100.0
	maxSleep := 10000.0

	temp := math.Min(maxSleep, base*math.Pow(float64(attempt), 2))
	sleep := time.Duration(temp/2+rand.Float64()*temp/2) * time.Millisecond
	debugf("failure to write into influxdb (attempt %d) sleeping for %s and retrying\n", attempt, sleep)
	time.Sleep(sleep)

	attempt++

	return writePoints(c, bp, attempt)
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

func logln(msg string) {
	_, _ = fmt.Fprintln(os.Stdout, msg)
}

func logf(msg string, a ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, msg, a...)
}

func handleError(err error) {
	if err != nil {
		_, logErr := fmt.Fprintf(os.Stderr, "%s\n", err)
		if logErr != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
