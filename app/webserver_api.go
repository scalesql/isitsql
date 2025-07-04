package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/scalesql/isitsql/internal/metricvaluering"
	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ChartDataSource struct {
	Series []ChartSeries `json:"series"`
}

type ChartDataSource2 struct {
	Series []ChartSeries2 `json:"series"`
}

type ChartSeries struct {
	Name string      `json:"name"`
	Data []ChartData `json:"data"`
}

type ChartSeries2 struct {
	Name string       `json:"name"`
	Data []ChartData2 `json:"data"`
}

type ChartData struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	// TS *time.Time `json:"ts,omitempty"`
}

type ChartData2 struct {
	X int64  `json:"x"`
	Y *int64 `json:"y"`
}

func ApiTest(w http.ResponseWriter, r *http.Request) {
	s1 := servers.Servers["S1"]
	json.NewEncoder(w).Encode(s1)
}

func ApiAll(w http.ResponseWriter, r *http.Request) {
	//log.Println("Called API all...")
	ss := servers.CloneAll()
	json.NewEncoder(w).Encode(ss)
}

func ApiServerJson(w http.ResponseWriter, r *http.Request) {
	server := r.PathValue("server")

	servers.RLock()
	s, ok := servers.Servers[server]
	servers.RUnlock()
	if ok {
		json.NewEncoder(w).Encode(s)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}
}

func ApiCpu(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.

	s := r.PathValue("server")
	// log.Println("API CPU  - ", s)

	servers.RLock()
	m, ok := servers.Servers[s]
	servers.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	// Get the CPU
	otherCpu := ChartSeries{Name: "Other CPU"}
	sqlCpu := ChartSeries{Name: "SQL CPU"}
	sqlSec := ChartSeries{Name: "SQL per Second"}

	//otherCpu = ChartSeries{Name: "Other CPU"}

	var v int64

	m.RLock()
	cpu := m.CPUUsage.Values()
	m.RUnlock()

	for i := 0; i < len(cpu); i++ {
		t := cpu[i].At.Unix() * 1000
		v = int64(cpu[i].Other)
		d := ChartData{X: t, Y: v}
		if d.X > 0 {
			otherCpu.Data = append(otherCpu.Data, d)
			sqlCpu.Data = append(sqlCpu.Data, ChartData{X: t, Y: int64(cpu[i].SQL)})
		}
	}
	var dataSource ChartDataSource
	dataSource.Series = append(dataSource.Series, otherCpu)
	dataSource.Series = append(dataSource.Series, sqlCpu)

	var allValues []*metricvaluering.MetricValue
	m.RLock()
	sqlMetric, sqlFound := m.Metrics["sql"]
	m.RUnlock()
	if sqlFound {
		allValues = sqlMetric.V2.Values()
		for i := 0; i < len(allValues); i++ {
			if allValues[i].ValuePerSecond > 0 {
				sqlSec.Data = append(sqlSec.Data,
					ChartData{
						X: allValues[i].EventTime.Unix() * 1000,
						Y: int64(allValues[i].ValuePerSecond)})
			}
		}

		dataSource.Series = append(dataSource.Series, sqlSec)

	}

	json.NewEncoder(w).Encode(dataSource)
}

// APIServerWaits servers up JSON waits
func APIServerWaits(w http.ResponseWriter, r *http.Request) {

	// Disable caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.

	key := r.PathValue("server")
	keepsort := r.URL.Query().Get("keepsort") == "1"

	// new waits ====================================
	start := time.Now()
	// results, err := globalWaitsBucket.ReadWaits(key)
	results, err := waitmap.ReadWaitFiles(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WinLogln(errors.Wrap(err, "apiserverwaits").Error())
		logrus.Error(err)
		return
	}
	// logrus.Infof("results: %d (%s)", len(results), time.Since(start))
	wr := WaitRing{}
	//start = time.Now()
	for _, w := range results {
		waits := w
		wr.Enqueue(&waits)
	}

	twg := wr.TopGroups()
	wv := wr.Values()
	if time.Since(start) > time.Duration(100*time.Millisecond) {
		logrus.Tracef("apiserverwaits: values: %d (%s)", len(wv), time.Since(start))
	}

	seriesCount := len(twg.SortedKeys)

	// cap this at five series
	if seriesCount > 5 {
		seriesCount = 5
	}

	if seriesCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	series := make([]ChartSeries, seriesCount)
	var i int
	for i = 0; i < seriesCount; i++ {
		series[i] = ChartSeries{Name: twg.SortedKeys[i]}
	}

	for i = 0; i < len(wv); i++ {
		for sn := 0; sn < seriesCount; sn++ {
			t := wv[i].EventTime.Unix() * 1000
			//fmt.Println(wv[i].EventTime)
			yval := wv[i].WaitSummary[twg.SortedKeys[sn]]
			yval = yval / 1000 // ms -> seconds
			d := ChartData{X: t, Y: yval}
			series[sn].Data = append(series[sn].Data, d)
		}
	}

	// the default is to reverse them, but "keepsort" will override that
	// and keep the original sort order
	var dataSource ChartDataSource
	if keepsort {
		dataSource.Series = append(dataSource.Series, series...)
	} else {
		for i = seriesCount - 1; i >= 0; i-- {
			dataSource.Series = append(dataSource.Series, series[i])
		}
	}

	json.NewEncoder(w).Encode(dataSource)
}

// APIServerWaits2 servers up JSON waits based on the realtime monitoring
func APIServerWaits2(w http.ResponseWriter, r *http.Request) {
	// println("waits2")
	// Disable caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.

	s := r.PathValue("server")
	keepsort := r.URL.Query().Get("keepsort") == "1"

	servers.RLock()
	_, ok := servers.Servers[s]
	servers.RUnlock()
	if !ok {
		//log.Println("ApitServerWaits: Map is empty")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Server Not Found"))
		logrus.Errorf("apiserverwaits2: not found: %s", s)
		return
	}
	// m.RLock()
	history := DynamicWaitRepository.Values(s)
	if len(history) == 0 {
		logrus.Tracef("apiserverwaits2: history: 0 items")
	}
	// m.RUnlock()

	// sm := m.WaitBox.History.Top(5)
	// if len(sm.BaseMap) == 0 {
	// 	logrus.Tracef("apiserverwaits2: sm.basemap: 0 items")
	// }
	sm := DynamicWaitRepository.Top(s, 5)
	if len(sm.SortedKeys) == 0 {
		logrus.Tracef("apiserverwaits2: sm.sortedkeys: 0 items")
	}
	seriesCount := len(sm.SortedKeys)
	if seriesCount == 0 {
		logrus.Tracef("apiserverwaits2: seriescount: 0 items")
	}

	// cap this at five series
	// if seriesCount > 5 {
	// 	seriesCount = 5
	// }
	// println(seriesCount)
	// removed since the server exists but with no data
	// if seriesCount == 0 {
	// 	w.WriteHeader(http.StatusNotFound)
	// 	w.Write([]byte("Series Not Found"))
	// 	//logrus.Errorf("apiserverwaits2: %s: no series", s)
	// 	return
	// }

	series := make([]ChartSeries, seriesCount)
	var i int
	for i = 0; i < seriesCount; i++ {
		series[i] = ChartSeries{Name: sm.SortedKeys[i]}
	}

	// just get the chosen series
	for i = 0; i < len(history); i++ {
		for sn := 0; sn < seriesCount; sn++ {
			t := history[i].TS.Unix() * 1000
			yval := history[i].Waits[sm.SortedKeys[sn]]
			yval = yval / 1000 // ms -> seconds
			d := ChartData{X: t, Y: yval}
			series[sn].Data = append(series[sn].Data, d)
		}
	}

	// the default is to reverse them, but "keepsort" will override that
	// and keep the original sort order
	var dataSource ChartDataSource
	if keepsort {
		dataSource.Series = append(dataSource.Series, series...)
	} else {
		for i = seriesCount - 1; i >= 0; i-- {
			dataSource.Series = append(dataSource.Series, series[i])
		}
	}

	json.NewEncoder(w).Encode(dataSource)
}

func ApiDisk(w http.ResponseWriter, r *http.Request) {

	// Disable caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.

	s := r.PathValue("server")
	//log.Println("API Disk  - ", s)

	servers.RLock()
	m, ok := servers.Servers[s]
	servers.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	// Get the Disk
	m.RLock()
	c1 := ChartSeries2{Name: "Disk Reads"}
	c1.GetChartData2(m.Metrics["bytesread"], (1024 * 1024))

	c2 := ChartSeries2{Name: "Disk Writes"}
	c2.GetChartData2(m.Metrics["byteswritten"], (1024 * 1024))

	c3 := ChartSeries2{Name: "Page Life Expectancy"}
	c3.GetChartData2(m.Metrics["ple"], 1)
	m.RUnlock()

	var dataSource ChartDataSource2
	dataSource.Series = append(dataSource.Series, c1)
	dataSource.Series = append(dataSource.Series, c2)
	dataSource.Series = append(dataSource.Series, c3)

	json.NewEncoder(w).Encode(dataSource)
}

// func (cs *ChartSeries) GetChartData(m Metric) error {
// 	// var v int64

// 	for i := 0; i < METRIC_ARRAY_SIZE; i++ {
// 		p := m.Pointer - i
// 		if p < 0 {
// 			p = p + METRIC_ARRAY_SIZE
// 		}

// 		if m.Accumulating {
// 			if m.Values[p].AggregateValue > 0 && m.Values[p].Value >= 0 && m.Values[p].EventTime.Unix() > 0 {
// 				cs.Data = append(cs.Data,
// 					ChartData{
// 						X: m.Values[p].EventTime.Unix() * 1000,
// 						Y: int64(m.Values[p].Value),
// 					})
// 			}
// 		} else {
// 			if m.Values[p].EventTime.Unix() > 0 {
// 				cs.Data = append(cs.Data,
// 					ChartData{
// 						X: m.Values[p].EventTime.Unix() * 1000,
// 						Y: int64(m.Values[p].Value),
// 					})
// 			}
// 		}
// 	}

// 	return nil
// }

func (cs *ChartSeries2) GetChartData2(m Metric, divisor int64) error {
	// var v int64

	if divisor == 0 {
		divisor = 1
	}

	//fmt.Println(m.V2.Values())
	allValues := m.V2.Values()

	//fmt.Println("**************************************************************")
	// for i := 0; i < len(allValues); i++ {
	// 	onev := allValues[i]
	// 	//fmt.Println(onev.EventTime, onev.PolledValue, onev.Value)
	// }

	var arrayValues [METRIC_ARRAY_SIZE]int64
	//var minTime int64
	//minTime = int64(^uint(0) >> 1)

	// for i := 0; i < METRIC_ARRAY_SIZE; i++ {
	// 	p := m.Pointer - i
	// 	if p < 0 {
	// 		p = p + METRIC_ARRAY_SIZE
	// 	}

	// 	// Populate the array with Value
	// 	// loop through and create ChartData2
	// 	// If PopulatedValue, use pointer to array, else nil

	// 	if m.Accumulating {
	// 		arrayValues[i] = m.Values[p].ValuePerSecond / divisor
	// 	} else {
	// 		arrayValues[i] = m.Values[p].Value / divisor
	// 	}

	// 	if m.Values[p].PolledValue {
	// 		cs.Data = append(cs.Data,
	// 			ChartData2{
	// 				X: m.Values[p].EventTime.Unix() * 1000,
	// 				Y: &arrayValues[i],
	// 			})
	// 	} else {
	// 		cs.Data = append(cs.Data,
	// 			ChartData2{
	// 				X: m.Values[p].EventTime.Unix() * 1000,
	// 				Y: nil,
	// 			})
	// 	}

	// }

	for i := 0; i < len(allValues); i++ {

		if m.Accumulating {
			arrayValues[i] = allValues[i].ValuePerSecond / divisor
		} else {
			arrayValues[i] = allValues[i].Value / divisor
		}

		if allValues[i].PolledValue {
			cs.Data = append(cs.Data,
				ChartData2{
					X: allValues[i].EventTime.Unix() * 1000,
					Y: &arrayValues[i],
				})
		} else {
			cs.Data = append(cs.Data,
				ChartData2{
					X: allValues[i].EventTime.Unix() * 1000,
					Y: nil,
				})
		}

	}

	return nil
}

// func ApiDates(w http.ResponseWriter, r *http.Request) {
// 	//log.Println("Got an API request...")
// 	// now := time.Now()

// 	// first series
// 	ms := time.Now().Unix() * 1000

// 	s1 := ChartSeries{Name: "Series-Bill"}
// 	servers.RLock()
// 	z := servers.Servers["S1"].Metrics["sql"]
// 	servers.RUnlock()

// 	var p int
// 	var d ChartData

// 	for i := 0; i < METRIC_ARRAY_SIZE; i++ {
// 		p = z.Pointer - i
// 		if p < 0 {
// 			p = p + METRIC_ARRAY_SIZE
// 		}
// 		if z.Values[p].Value > 0 {
// 			t := z.Values[p].EventTime.Unix() * 1000
// 			v := z.Values[p].Value
// 			d = ChartData{X: t, Y: v}
// 			s1.Data = append(s1.Data, d)
// 		}

// 	}

// 	// 	d := ChartData{ms, 4}
// 	//
// 	// 	s1.Data = append(s1.Data, d)
// 	//
// 	// 	d = ChartData{ms + 60000, 7}
// 	// 	s1.Data = append(s1.Data, d)
// 	//
// 	// 	d = ChartData{ms + 1800000, 2}
// 	// 	s1.Data = append(s1.Data, d)

// 	s2 := ChartSeries{Name: "Woot!"}
// 	d = ChartData{ms + 30000, 22}
// 	s2.Data = append(s2.Data, d)

// 	d = ChartData{ms + 1100000, 19}
// 	s2.Data = append(s2.Data, d)

// 	var dataSource ChartDataSource
// 	dataSource.Series = append(dataSource.Series, s1)
// 	dataSource.Series = append(dataSource.Series, s2)

// 	json.NewEncoder(w).Encode(dataSource)

// }
