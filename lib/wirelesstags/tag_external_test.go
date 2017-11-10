package wirelesstags_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"strings"

	"github.com/stojg/grabber/lib/wirelesstags"
)

func TestGet(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "GetTagList2") {
			http.ServeFile(w, r, "testdata/GetTagList2.json")
		}
		if strings.Contains(r.URL.String(), "LoadTempSensorConfig") {
			http.ServeFile(w, r, "testdata/LoadTempSensorConfig.json")
		}
		if strings.Contains(r.URL.String(), "GetMultiTagStatsRaw") {
			http.ServeFile(w, r, "testdata/GetMultiTagStatsRaw_temperature.json")
		}
	}))
	defer ts.Close()

	client := &http.Client{}
	wt := wirelesstags.New(client, ts.URL, "")
	tags, err := wt.Get(time.Now())
	if err != nil {
		t.Error(err)
		return
	}
	expected := 10
	actual := len(tags)
	if actual != expected {
		t.Errorf("Expected %d tags, got %d tags", actual, expected)
		return
	}
}
