package main

import (
	"fmt"
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

	lastUpdated := time.Now().In(loc).Add(-2 * 24 * time.Hour)
	for ticker := time.NewTicker(time.Minute * 5); true; <-ticker.C {
		//logf("fetching data from %s\n", lastUpdated)
		if err := update(tagClient, influxClient, cfg.InfluxDB.DB, lastUpdated); err != nil {
			logf("error: %s\n", err)
		}
		lastUpdated = time.Now().In(loc)
	}

}

func update(wirelessTags *wirelesstags.Client, influxClient influx.Client, databaseName string, fromTime time.Time) error {

	tags, err := wirelessTags.Get(fromTime)
	if err != nil {
		return fmt.Errorf("wirelessTags.Get - %v", err)
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

	// @todo this looks weird, looks like it doesnt batch
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
