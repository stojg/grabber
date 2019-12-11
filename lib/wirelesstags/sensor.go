package wirelesstags

import (
	"fmt"
	"strings"
)

// Sensor represents a single sensor tag with the information provided from the API. The metrics can be found at Sensor.Metrics
type Sensor struct {
	ManagerName         string  `json:"managerName"`
	Mac                 string  "json:mac"
	Name                string  `json:"name"`
	UUID                string  `json:"uuid"`
	Comment             string  `json:"comment"`
	SlaveID             uint8   `json:"slaveId"`
	TagType             int     `json:"tagType"`
	LastComm            int64   `json:"lastComm"`
	Alive               bool    `json:"alive"`
	SignaldBm           int     `json:"signaldBm"`
	BatteryVolt         float64 `json:"batteryVolt"`
	Beeping             bool    `json:"beeping"`
	Lit                 bool    `json:"lit"`
	MigrationPending    bool    `json:"migrationPending"`
	BeepDurationDefault int     `json:"beepDurationDefault"`
	//EventState          int     `json:"eventState"`
	//TempEventState int `json:"tempEventState"` // Disarmed, TooLow, TooHigh, Normal
	OutOfRange    bool    `json:"outOfRange"`
	Lux           float64 `json:"lux"`
	Temperature   float64 `json:"temperature"`
	TempCalOffset float64 `json:"tempCalOffset"`
	CapCalOffset  int     `json:"capCalOffset"`
	// image_md5
	Humidity float64 `json:"cap"`
	// CapRaw           int     `json:"capRaw"` //0??
	// Az2              int     `json:"az2"` // no idea
	// CapEventState    int     `json:"capEventState"` // no idea
	// LightEventState  int     `json:"lightEventState"` // no idfea
	// Shorted          bool    `json:"shorted"`// but why?
	// zmod
	// thermostat
	// playback
	// PostBackInterval int     `json:"postBackInterval"`
	//Rev              int     `json:"rev"`
	//Version1         int     `json:"version1"`
	//FreqOffset       int     `json:"freqOffset"` ??
	//FreqCalApplied   int     `json:"freqCalApplied"` >>
	//ReviveEvery      int     `json:"reviveEvery"` ??
	//OorGrace         int     `json:"oorGrace"` ??
	//LBTh             float64 `json:"LBTh"`
	//EnLBN            bool    `json:"enLBN"`
	//Txpwr            int     `json:"txpwr"`
	//RssiMode         bool    `json:"rssiMode"`
	//Ds18             bool    `json:"ds18"`
	//BatteryRemaining float64 `json:"batteryRemaining"` weird float, percent?

	Metrics MetricsCollection
}

// Labels returns a map of key / value. 'name' and 'id' is always returned. The extra labels are added to the comment
// field in the UI and follows the name1=value1,name2=value2 format. It should be relatively resilient to whitespaces.
func (s *Sensor) Labels() map[string]string {
	labels := make(map[string]string)
	labels["name"] = s.Name
	labels["id"] = fmt.Sprintf("%d", s.SlaveID)
	extraLabels := strings.Split(s.Comment, ",")
	for _, extra := range extraLabels {
		keyValues := strings.Split(extra, "=")
		if len(keyValues) != 2 {
			continue
		}
		key := strings.Trim(keyValues[0], " ")
		labels[key] = strings.Trim(keyValues[1], " ")
	}
	return labels
}

func (s *Sensor) hasMotionSensor() bool {
	return inArray(s.TagType, []int{12, 13, 21})
}

func (s *Sensor) hasLightSensor() bool {
	return inArray(s.TagType, []int{26})
}

func (s *Sensor) hasMoistureSensor() bool {
	return inArray(s.TagType, []int{32, 33})
}

func (s *Sensor) hasWaterSensor() bool {
	return inArray(s.TagType, []int{32, 33})
}

func (s *Sensor) hasReedSensor() bool {
	return inArray(s.TagType, []int{52, 53})
}

func (s *Sensor) hasPIRSensor() bool {
	return inArray(s.TagType, []int{72})
}

func (s *Sensor) hasEventSensor() bool {
	return s.hasMotionSensor() || s.hasLightSensor() || s.hasReedSensor() || s.hasPIRSensor()
}

func (s *Sensor) hasHumiditySensor() bool {
	return s.hasHTU()
}

func (s *Sensor) hasTempSensor() bool {
	return !inArray(s.TagType, []int{82, 92})
}

func (s *Sensor) hasCurrentSensor() bool {
	return s.TagType == 42
}

/** Whether the tag's temperature sensor is high-precision (> 8-bit). */
func (s *Sensor) hasHTU() bool {
	return inArray(s.TagType, []int{13, 21, 52, 26, 72})
}

// can playback data that was recorded while being offline
func (s *Sensor) canPlayback() bool {
	return s.TagType == 21
}

func inArray(needle int, haystack []int) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}
