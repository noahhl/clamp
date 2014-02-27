package clamp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Stat struct {
	Key   string
	Value string
}

var statsChan chan Stat

func StartStatsServer(statsAddr string) chan Stat {
	statsChan = make(chan Stat, 100)
	stats := make(map[string]string, 10)
	stats["startupTime"] = time.Now().String()

	go func() {
		for stat := range statsChan {
			stats[stat.Key] = stat.Value
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(stats)
		fmt.Fprintf(w, "%v", string(b))
	})

	go http.ListenAndServe(statsAddr, nil)
	fmt.Printf("%v: Started stats server\n", time.Now())
	return statsChan
}