package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/stojg/grabber/lib/wirelesstags"
)

var (
	netClient  *http.Client
	pool       *x509.CertPool
	cpuProfile = flag.String("cpu-profile", "", "write cpu profile to file")
	memProfile = flag.String("mem-profile", "", "write memory profile to this file")
)

const (
	maxBatch = 1000
)

func init() {
	pool = x509.NewCertPool()
	pool.AppendCertsFromPEM(pemCerts)
	netClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

func main() {

	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	loc, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TZ load location error: %s\n", err)
		os.Exit(1)
	}

	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("['sys_mb': %0.2f,'numGC': %d]\n", float64(m.Sys)/1024/1024, m.NumGC)
			time.Sleep(1 * time.Second)
		}
	}()

	lastUpdated := time.Now().In(loc).Add(-24 * time.Hour * 1)
	if err := update(lastUpdated, loc); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}
	lastUpdated = time.Now().In(loc)

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		fmt.Printf("wrote memory profile to %s\n", *memProfile)
		return
	}

	ticker := time.NewTicker(time.Minute * 1)
	for range ticker.C {
		if err := update(lastUpdated, loc); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		lastUpdated = time.Now().In(loc)
	}
}

func update(lastUpdated time.Time, location *time.Location) error {

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
		return fmt.Errorf("error on tag update: %v", err)
	}

	fmt.Printf("read metrics for %d tags from mytaglist.com\n", len(tags))

	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     influxHost,
		Username: influxUser,
		Password: influxPassword,
	})

	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	// Create a new point batch
	bp := getNewPointBatch(influxDB)

	y := 0
	for _, tag := range tags {
		for unixTime, metrics := range tag.Metrics {
			wrote, err := addPoint(c, bp, tag.Labels(), metrics, unixTime)
			if err != nil {
				return err
			}
			y++
			if wrote {
				bp = getNewPointBatch(influxDB)
			}
		}
	}

	err = writePoints(c, bp)
	fmt.Fprintf(os.Stdout, "Updated %d data points\n", y)
	return err
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

	if len(bp.Points()) >= maxBatch {
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
