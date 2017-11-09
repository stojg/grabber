package wirelesstags

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGet(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "GetMultiTagStatsRaw") {
			http.ServeFile(w, r, "testdata/GetMultiTagStatsRaw_temperature.json")
		}
	}))
	defer ts.Close()

	metrics, err := Stats(ts.URL, []int{3, 1}, time.Date(2017, time.October, 15, 15, 0, 0, 0, time.Local))
	if err != nil {
		t.Error(err)
		return
	}

	for id, ids := range metrics {
		for ts, metric := range ids {
			for key, value := range metric {
				if _, ok := metric[key]; !ok {
					continue
				}
				fmt.Println(id, ts.Format(time.StampMilli), key, value)
			}

		}
	}

	if len(metrics) != 2 {
		t.Errorf("Expected %d tag metrics, got %d", 10, len(metrics))
		return
	}

	if len(metrics[1]) != 42*2 {
		t.Errorf("Expected %d metrics for tag 1, got %d", 42*2, len(metrics[1]))
		return
	}

	if len(metrics[3]) != 38*2 {
		t.Errorf("Expected %d metrics for tag 3, got %d", 38*2, len(metrics[3]))
		return
	}

	//if data == nil {
	//	t.Errorf("Expected data, got nil tags")
	//}

	// {'ids': [1], 'type': 'temperature', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'light', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'cap', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
	// {'ids': [1], 'type': 'motion', 'fromDate': '10/15/2017', 'toDate': '10/16/2017'}
}
