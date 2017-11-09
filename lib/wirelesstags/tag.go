package wirelesstags

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	TYPE_TEMPERATURE = "temperature"
	TYPE_LUX         = "light"
	TYPE_HUMIDITY    = "cap"
	TYPE_MOTION      = "motion"
	TYPE_BATTERY     = "batteryVolt"
	TYPE_SIGNAL      = "signal"
)

func Get(token string, domain string, since time.Time) ([]*Tag, error) {

	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	var body = []byte(`{}`)
	req, err := http.NewRequest("POST", domain+"/ethClient.asmx/GetTagList2", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer " + token)
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}

	resp, err := netClient.Do(req)
	if err != nil {
		return make([]*Tag, 0), err
	}
	defer resp.Body.Close()


	if resp.StatusCode != 200 {
	    return nil, fmt.Errorf("Got status code %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&result); err != nil {
		return make([]*Tag, 0), fmt.Errorf("Error parsing json response %v", err)
	}

	var tags []*Tag
	if err := mapstructure.Decode(result["d"], &tags); err != nil {
		return nil, fmt.Errorf("Error while decoding tag data: %v", err)
	}

	var temperatureTags []int
	var lightTags []int
	var humidityTags []int

	for _, t := range tags {

		lastConn := windowFILETIME(t.LastComm)

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

	metrics := make(map[int]MetricsCollection)
	if err := getMetric(token, domain, temperatureTags, TYPE_TEMPERATURE, metrics, since); err != nil {
		return nil, err
	}

	if err := getMetric(token, domain, humidityTags, TYPE_HUMIDITY, metrics, since); err != nil {
		return nil, err
	}

	if err := getMetric(token, domain, lightTags, TYPE_LUX, metrics, since); err != nil {
		return nil, err
	}

	for _, tag := range tags {
		if m, ok := metrics[tag.SlaveID]; ok {
			tag.Metrics = m
		}
	}

	return tags, err
}

func windowFILETIME(a int64) time.Time {
	// Windows FILETIME is 100 nanosecond intervals since January 1, 1601 (UTC)
	// Unix Date time is seconds since January 1, 1970 (UTC)
	// Offset between the two epochs in milliseconds is 11644473600000
	return time.Unix((a/10000-(11644473600000))/1000, 0)
}

func NiceType(t string) string {
	switch t {
	case TYPE_TEMPERATURE:
		return "temperature"
	case TYPE_LUX:
		return "lux"
	case TYPE_HUMIDITY:
		return "humidity"
	case TYPE_MOTION:
		return "motion"
	case TYPE_BATTERY:
		return "battery"
	case TYPE_SIGNAL:
		return "signal"
	default:
		return t
	}
}

func getMetric(token, domain string, ids []int, metricType string, metrics map[int]MetricsCollection, since time.Time) error {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	input := &GetMultiTagStatsRawInput{
		IDs:      ids,
		Type:     metricType,
		FromDate: since.Format("1/2/2006"),
		ToDate:   time.Now().Format("1/2/2006"),
	}

	requestBody, err := json.Marshal(input)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", domain+"/ethLogs.asmx/GetMultiTagStatsRaw", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer " + token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := netClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		var message Message
		if err := json.Unmarshal(b, &message); err != nil {
			return err
		}
		return fmt.Errorf("%s", message.Message)
	}

	var result map[string]struct {
		Stats    RawStats `json:"stats"`
		TempUnit int      `json:"temp_unit"`
		Ids      []int    `json:"ids"`
		Names    []string `json:"names"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return fmt.Errorf("Error decoding json response for Data %v", err)
	}

	auckland, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
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

				metrics[slaveID][timestamp][NiceType(metricType)] = stat.Values[i][j]
			}
		}
	}

	return nil
}

type MetricsCollection map[time.Time]Metric

type GetMultiTagStatsRawInput struct {
	IDs      []int  `json:"ids"`
	Type     string `json:"type"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

type RawStats []struct {
	Date   string      `json:"date"`
	Ids    []int       `json:"ids"`
	Values [][]float64 `json:"values"`
	Tods   [][]int     `json:"tods"`
}

type Metric map[string]interface{}

type Message struct {
	Message       string
	ExceptionType string
	StackTrace    string
}

type Tag struct {
	Name           string  `json:"name"`
	Comment        string  `json:"comment"`
	TempEventState int     `json:"tempEventState"` // Disarmed, TooLow, TooHigh, Normal
	OutOfRange     bool    `json:"outOfRange"`
	Temperature    float64 `json:"temperature"`
	SlaveID        int     `json:"slaveId"`
	BatteryVolt    float64 `json:"batteryVolt"`
	Lux            float64 `json:"lux"`
	Humidity       float64 `json:"cap"` // humidity
	Type           string  `json:"__type"`
	ManagerName    string  `json:"managerName"`
	//Mac                 string        `json:"mac"`
	//Dbid                int           `json:"dbid"`
	//Mirrors             []interface{} `json:"mirrors"`
	//NotificationJS      string        `json:"notificationJS"`
	//UUID                string        `json:"uuid"`
	TagType             int     `json:"tagType"`
	LastComm            int64   `json:"lastComm"`
	Alive               bool    `json:"alive"`
	SignaldBm           int     `json:"signaldBm"`
	Beeping             bool    `json:"beeping"`
	Lit                 bool    `json:"lit"`
	MigrationPending    bool    `json:"migrationPending"`
	BeepDurationDefault int     `json:"beepDurationDefault"`
	EventState          int     `json:"eventState"`
	TempCalOffset       float64 `json:"tempCalOffset"`
	CapCalOffset        int     `json:"capCalOffset"`
	CapRaw              int     `json:"capRaw"`
	Az2                 int     `json:"az2"`
	CapEventState       int     `json:"capEventState"`
	LightEventState     int     `json:"lightEventState"`
	Shorted             bool    `json:"shorted"`
	PostBackInterval    int     `json:"postBackInterval"`
	Rev                 int     `json:"rev"`
	Version1            int     `json:"version1"`
	FreqOffset          int     `json:"freqOffset"`
	FreqCalApplied      int     `json:"freqCalApplied"`
	ReviveEvery         int     `json:"reviveEvery"`
	OorGrace            int     `json:"oorGrace"`
	LBTh                float64 `json:"LBTh"`
	EnLBN               bool    `json:"enLBN"`
	Txpwr               int     `json:"txpwr"`
	RssiMode            bool    `json:"rssiMode"`
	Ds18                bool    `json:"ds18"`
	BatteryRemaining    float64 `json:"batteryRemaining"`

	Metrics MetricsCollection
}

func (t *Tag) hasMotionSensor() bool {
	return inArray(t.TagType, []int{12, 13, 21})
}

func (t *Tag) hasLightSensor() bool {
	return inArray(t.TagType, []int{26})
}

func (t *Tag) hasMoistureSensor() bool {
	return inArray(t.TagType, []int{32, 33})
}

func (t *Tag) hasWaterSensor() bool {
	return inArray(t.TagType, []int{32, 33})
}

func (t *Tag) hasReedSensor() bool {
	return inArray(t.TagType, []int{52, 53})
}

func (t *Tag) hasPIRSensor() bool {
	return inArray(t.TagType, []int{72})
}

func (t *Tag) hasEventSensor() bool {
	return t.hasMotionSensor() || t.hasLightSensor() || t.hasReedSensor() || t.hasPIRSensor()
}

func (t *Tag) hasHumiditySensor() bool {
	return t.hasHTU()
}

func (t *Tag) hasTempSensor() bool {
	return !inArray(t.TagType, []int{82, 92})
}

func (t *Tag) hasCurrentSensor() bool {
	return t.TagType == 42
}

/** Whether the tag's temperature sensor is high-precision (> 8-bit). */
func (t *Tag) hasHTU() bool {
	return inArray(t.TagType, []int{13, 21, 52, 26, 72})
}

// can playback data that was recorded while being offline
func (t *Tag) canPlayback() bool {
	return t.TagType == 21
}

func inArray(needle int, haystack []int) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}
