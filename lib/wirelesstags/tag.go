package wirelesstags

// Tag represents a single sensor tag with the information provided from the API. The metrics can be found at Tag.Metrics
type Tag struct {
	Name                string  `json:"name"`
	Comment             string  `json:"comment"`
	TempEventState      int     `json:"tempEventState"` // Disarmed, TooLow, TooHigh, Normal
	OutOfRange          bool    `json:"outOfRange"`
	Temperature         float64 `json:"temperature"`
	SlaveID             int     `json:"slaveId"`
	BatteryVolt         float64 `json:"batteryVolt"`
	Lux                 float64 `json:"lux"`
	Humidity            float64 `json:"cap"` // humidity
	Type                string  `json:"__type"`
	ManagerName         string  `json:"managerName"`
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
