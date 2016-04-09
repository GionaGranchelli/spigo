// Collect throughput and response times using Go-Kit Metrics
package collect

import (
	"fmt"
	. "github.com/adrianco/goguesstimate/guesstimate"
	"github.com/adrianco/spigo/archaius"
	"github.com/adrianco/spigo/names"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/expvar"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	maxHistObservable = 1000000 // one millisecond
	sampleCount       = 1000    // data points will be sampled 5000 times to build a distribution by guesstimate
)

//var mon = monitor.GetMonitors()
//save a sample of the actual data for use by guesstimate
var sampleMap map[metrics.Histogram][]int64

func NewHist(name string) metrics.Histogram {
	var h metrics.Histogram
	if name != "" && archaius.Conf.Collect {
		h = expvar.NewHistogram(name, 1000, maxHistObservable, 1, []int{50, 99}...)
		if sampleMap == nil {
			sampleMap = make(map[metrics.Histogram][]int64)
		}
		sampleMap[h] = make([]int64, 0, sampleCount)
		return h
	}
	return nil
}

func Measure(h metrics.Histogram, d time.Duration) {
	if h != nil && archaius.Conf.Collect {
		if d > maxHistObservable {
			h.Observe(int64(maxHistObservable))
		} else {
			h.Observe(int64(d))
		}
		s := sampleMap[h]
		if s != nil && len(s) < sampleCount {
			sampleMap[h] = append(s, int64(d))
		}
	}
}

// have to pass in name because metrics.Histogram blocks expvar.Historgram.Name()
func SaveHist(h metrics.Histogram, name, suffix string) {
	if archaius.Conf.Collect {
		file, err := os.Create("csv_metrics/" + names.Arch(name) + "_" + names.Machine(name) + suffix + ".csv")
		if err != nil {
			log.Printf("%v: %v\n", name, err)
		}
		metrics.PrintDistribution(file, h)
		file.Close()
	}
}

func SaveAllGuesses(name string) {
	if len(sampleMap) == 0 {
		return
	}
	log.Printf("Saving %v histograms for Guesstimate\n", len(sampleMap))
	var g Guess
	g = Guess{
		Space: GuessModel{
			Name:        names.Arch(name),
			Description: "Guesstimate generated by github.com/adrianco/spigo",
			IsPrivate:   "true",
			Graph: GuessGraph{
				Metrics:      make([]GuessMetric, 0, len(sampleMap)),
				Guesstimates: make([]Guesstimate, 0, len(sampleMap)),
			},
		},
	}
	row := 1
	col := 1
	seq := []string{"", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	for h, data := range sampleMap {
		g.Space.Graph.Metrics = append(g.Space.Graph.Metrics, GuessMetric{
			ID:         seq[row] + seq[col],
			ReadableID: seq[row] + seq[col],
			Name:       h.Name(),
			Location:   GuessMetricLocation{row, col},
		})
		g.Space.Graph.Guesstimates = append(g.Space.Graph.Guesstimates, Guesstimate{
			Metric:          seq[row] + seq[col],
			Input:           "",
			GuesstimateType: "DATA",
			Data:            data,
		})
		row++
		if row >= len(seq) {
			row = 1
			col++
			if col >= len(seq) {
				break
			}
		}
	}
	SaveGuess(g, "json_metrics/"+names.Arch(name))
}

func Save() {
	//	if archaius.Conf.Collect {
	//		file, _ := os.Create("csv_metrics/" + archaius.Conf.Arch + "_metrics.csv")
	//		counters, gauges := metrics.Snapshot()
	//		cj, _ := json.Marshal(counters)
	//		gj, _ := json.Marshal(gauges)
	//		file.WriteString(fmt.Sprintf("{\n\"counters\":%v\n\"gauges\":%v\n}\n", string(cj), string(gj)))
	//		file.Close()
	//	}
}

func Serve(port int) {
	sock, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("HTTP metrics now available at localhost:%v/debug/vars", port)
		http.Serve(sock, nil)
	}()
}
