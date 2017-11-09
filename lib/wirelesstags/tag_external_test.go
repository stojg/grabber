package wirelesstags_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	}))
	defer ts.Close()

	tags, err := wirelesstags.Get(ts.URL)
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

	// {'ids': [1], 'type': 'temperature', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'light', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'cap', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'motion', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
}
