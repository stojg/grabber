package wirelesstags

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"io"

	"github.com/mitchellh/mapstructure"
)

const (
	typeTemperature = "temperature"
	typeLux         = "light"
	typeHumidity    = "cap"
	typeMotion      = "motion"
	typeBattery     = "batteryVolt"
	typeSignal      = "signal"
)

func typeToString(t string) string {
	switch t {
	case typeTemperature:
		return "temperature"
	case typeLux:
		return "lux"
	case typeHumidity:
		return "humidity"
	case typeMotion:
		return "motion"
	case typeBattery:
		return "battery"
	case typeSignal:
		return "signal"
	default:
		return t
	}
}

// New creates a new client that is used for fetching tag sensor information from http://www.wirelesstag.net/
func New(client *http.Client, domain, token string, location *time.Location) *WirelessTags {
	return &WirelessTags{
		client:   client,
		domain:   domain,
		token:    token,
		location: location,
	}
}

// WirelessTags is a holder for information used for getting and parsing sensor tag data
type WirelessTags struct {
	client   *http.Client
	domain   string
	token    string
	location *time.Location
}

// Get all sensor data and return a list of Sensor
func (w *WirelessTags) Get(since time.Time) ([]*Sensor, error) {

	var body = []byte(`{}`)
	req, err := http.NewRequest("POST", w.domain+"/ethClient.asmx/GetTagList2", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}

	var resp *http.Response
	if resp, err = w.client.Do(req); err != nil {
		return make([]*Sensor, 0), fmt.Errorf("error during tag GetTagList2: %v", err)
	}
	defer closer(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("got status code %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	if err = dec.Decode(&result); err != nil {
		return make([]*Sensor, 0), fmt.Errorf("error parsing json response %v", err)
	}

	var tags []*Sensor
	if err = mapstructure.Decode(result["d"], &tags); err != nil {
		return nil, fmt.Errorf("error while decoding tag data: %v", err)
	}

	var temperatureTags []int
	var lightTags []int
	var humidityTags []int

	for _, t := range tags {

		lastConn := windowsFileTime(t.LastComm)

		if lastConn.Before(since) {
			since = lastConn
		}

		if t.hasTempSensor() {
			temperatureTags = append(temperatureTags, t.SlaveID)
		}
		if t.hasLightSensor() {
			lightTags = append(lightTags, t.SlaveID)
		}
		if t.hasHumiditySensor() {
			humidityTags = append(humidityTags, t.SlaveID)
		}
	}

	// the metrics are keyed by the sensors slaveID
	metrics := make(map[int]MetricsCollection)
	if err = w.updateMetrics(temperatureTags, typeTemperature, metrics, since); err != nil {
		return nil, err
	}

	if err = w.updateMetrics(humidityTags, typeHumidity, metrics, since); err != nil {
		return nil, err
	}

	if err = w.updateMetrics(lightTags, typeLux, metrics, since); err != nil {
		return nil, err
	}

	for _, tag := range tags {
		if m, ok := metrics[tag.SlaveID]; ok {
			tag.Metrics = m
		}
	}

	return tags, err
}

// windowsFileTime returns the windows FILETIME value in Unix time.
//  - Windows FILETIME is 100 nanosecond intervals since January 1, 1601 (UTC)
//  - Unix Date time is seconds since January 1, 1970 (UTC)
//  - Offset between the two epochs in milliseconds is 116444736e+5
// Note that the smallest return resolution is milliseconds
func windowsFileTime(intervals int64) time.Time {
	// we need to convert 100ns intervals to ms so we don't overflow on int64
	var ms = intervals / 10 / 1000

	// offset between windows epoch and unix epoch in milliseconds
	var epochOffset int64 = 116444736e+5

	// millisecond since unix epoch start
	var unix = time.Millisecond * time.Duration(ms-epochOffset)

	sec := unix / time.Second
	nsec := unix % time.Second

	return time.Unix(int64(sec), int64(nsec))
}

func (w *WirelessTags) updateMetrics(ids []int, metricType string, metrics map[int]MetricsCollection, since time.Time) error {

	if len(ids) == 0 {
		return nil
	}

	var resp *http.Response
	var err error
	if resp, err = w.requestMetrics(ids, metricType, since); err != nil {
		return err
	}

	defer closer(resp.Body)

	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var message message
		if err = json.Unmarshal(body, &message); err != nil {
			return err
		}
		return fmt.Errorf("%s", message.Message)
	}

	var result map[string]struct {
		Stats    rawStats `json:"stats"`
		TempUnit int      `json:"temp_unit"`
		Ids      []int    `json:"ids"`
		Names    []string `json:"names"`
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error decoding JSON response %s: %v", resp.Request.URL, err)
	}

	var auckland *time.Location
	if auckland, err = time.LoadLocation("Pacific/Auckland"); err != nil {
		return err
	}

	for _, stat := range result["d"].Stats {
		startDate, err := time.ParseInLocation("1/2/2006", stat.Date, auckland)
		if err != nil {
			return fmt.Errorf("Can't parse start date %s", stat.Date)
		}
		for i, slaveID := range stat.Ids {
			for j := range stat.Tods[i] {
				timestamp := startDate.Add(time.Second * time.Duration(stat.Tods[i][j]))
				if timestamp.Before(since) {
					continue
				}

				if _, ok := metrics[slaveID]; !ok {
					metrics[slaveID] = make(map[time.Time]Metric)
				}

				if _, ok := metrics[slaveID][timestamp]; !ok {
					metrics[slaveID][timestamp] = Metric{}
				}

				metrics[slaveID][timestamp][typeToString(metricType)] = stat.Values[i][j]
			}
		}
	}

	return nil
}

func (w *WirelessTags) requestMetrics(ids []int, metricType string, since time.Time) (*http.Response, error) {
	input := &getMultiTagStatsRawInput{
		IDs:      ids,
		Type:     metricType,
		FromDate: since.Format("1/2/2006"),
		ToDate:   time.Now().In(w.location).Format("1/2/2006"),
	}

	requestBody, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", w.domain+"/ethLogs.asmx/GetMultiTagStatsRaw", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Content-Type", "application/json")

	return w.client.Do(req)
}

// Metric are typically something like this metric["temperature"] = 19.0
type Metric map[string]interface{}

// MetricsCollection buckets metrics in a timestamp for a more efficient updating of the metrics in the backend
type MetricsCollection map[time.Time]Metric

type getMultiTagStatsRawInput struct {
	IDs      []int  `json:"ids"`
	Type     string `json:"type"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

type rawStats []struct {
	Date   string      `json:"date"`
	Ids    []int       `json:"ids"`
	Values [][]float64 `json:"values"`
	Tods   [][]int     `json:"tods"`
}

type message struct {
	Message       string
	ExceptionType string
	StackTrace    string
}

func closer(c io.Closer) {
	err := c.Close()
	if err != nil {
		fmt.Printf("Error during Close: err")
	}
}
