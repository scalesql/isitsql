package app

import (
	"database/sql"
	"time"

	"github.com/scalesql/isitsql/internal/metricvaluering"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

// GetMetric sets a single metric value
func (s *SqlServerWrapper) GetMetric(metric, stmt string, accumulating bool) error {

	var m metricvaluering.MetricValue
	s.RLock()
	db := s.DB
	reset := s.ResetOnThisPoll
	s.RUnlock()

	// TODO: Show gaps: http://stackoverflow.com/questions/14821269/show-x-axis-gaps-highstocks
	// db, err := sql.Open(s.ConnectionType, s.ConnectionString)
	// db.SetConnMaxLifetime(30 * time.Second)
	// if err != nil {
	// 	return err
	// }
	// defer db.Close()

	// v is a copy of the metric.  It doesn't need locking
	s.RLock()
	v, ok := s.Metrics[metric]
	s.RUnlock()

	// if this metric doesn't exist, populate empty values with real time stamps
	// Don't populate this for now
	if !ok {
		ticker := time.Now().Add(-60 * time.Minute)
		for i := 0; i < METRIC_ARRAY_SIZE; i++ {
			mv := metricvaluering.MetricValue{
				EventTime:      ticker,
				PolledValue:    false,
				Value:          0,
				ValuePerSecond: 0,
				AggregateValue: 0}

			s.Lock()
			v.V2.Enqueue(&mv)
			s.Unlock()
			ticker = ticker.Add(1 * time.Minute)
			//fmt.Println(metric, mv.EventTime)
		}
	}

	v.Accumulating = accumulating
	previous := v.V2.GetLastValue() // get last written value

	// increment the pointer
	// v.Pointer = v.Pointer + 1
	// if v.Pointer > METRIC_ARRAY_SIZE-1 {
	// 	v.Pointer = 0
	// }

	// In case we fail to poll, set some default values
	m.PolledValue = false
	m.EventTime = time.Now()

	row := db.QueryRow(stmt)
	err := row.Scan(&m.AggregateValue)
	if err != nil {
		// if our polling failed, put it back with a default value
		s.Lock()
		v.V2.Enqueue(&m)
		s.Metrics[metric] = v
		s.Unlock()
		if err == sql.ErrNoRows { // swallow missing performance counters
			return nil
		}
		return errors.Wrap(err, "scan")
	}
	m.PolledValue = true

	if accumulating {
		// if there's a previous value and that value is increasing and we aren't resetting
		// figure out all the data stuff
		if previous.AggregateValue != 0 && m.AggregateValue > previous.AggregateValue && !reset {
			m.Value = m.AggregateValue - previous.AggregateValue
			m.DeltaDuration = m.EventTime.Sub(previous.EventTime)

			// if we can, compute a value per second
			// I really should convert value to float64, then divide, then back to int64
			var seconds = int64(m.DeltaDuration.Seconds())
			if m.Value != 0 && m.DeltaDuration != 0 && seconds > 0 {
				m.ValuePerSecond = m.Value / seconds
				//log.Printf("%s %d %d", metric, m.ValuePerSecond, int64(seconds))
			} else {
				m.PolledValue = false
				m.ValuePerSecond = 0
			}
			// there isn't a previous value or we are resetting or this value is less than the previous value
			// Just reset the aggregates so we'll be right the next time
		} else {
			m.Value = 0
			m.DeltaDuration = m.EventTime.Sub(previous.EventTime)
		}
		// not accumulating
	} else {
		m.Value = m.AggregateValue
		m.DeltaDuration = m.EventTime.Sub(previous.EventTime)
	}

	// v is a copy of the metric.  It doesn't need locking
	v.V2.Enqueue(&m)

	s.Lock()
	s.Metrics[metric] = v
	s.LastPollTime = time.Now()
	s.Unlock()

	return nil
}

func (s *SqlServerWrapper) GetLastMetric(metric string) (*metricvaluering.MetricValue, error) {
	var e metricvaluering.MetricValue

	//servers.RLock()
	s.RLock()
	v, ok := s.Metrics[metric]
	s.RUnlock()
	//servers.RUnlock()
	if !ok {
		return &e, errors.New("Invalid metric")
	}

	m := v.V2.GetLastValue()
	if m == nil {
		return m, errors.New("GetLastMetric: No metric found")
	}

	return m, nil

	// if v.Pointer > len(v.Values) {
	// 	var e MetricValue
	// 	return e, errors.New("Invalid array pointer")
	// }

	// m := v.Values[v.Pointer]

	// return m, nil
}

func (s *SqlServerWrapper) GetLastPLE() int64 {
	m, err := s.GetLastMetric("ple")
	if err != nil {
		return 0
	}
	return m.Value
}

func (s *SqlServerWrapper) GetLastPLEString() string {
	return humanize.Comma(s.GetLastPLE())
}

func (s *SqlServerWrapper) GetLastSqlPerSecond() int64 {
	m, err := s.GetLastMetric("sql")
	if err != nil {
		return 0
	}
	return m.ValuePerSecond
}

func (s *SqlServerWrapper) GetLastSqlPerSecondString() string {
	return humanize.Comma(s.GetLastSqlPerSecond())
}

//////////////////////////////////////
// Local versions

func (s *SqlServer) GetLastMetric(metric string) (*metricvaluering.MetricValue, error) {
	var e metricvaluering.MetricValue

	v, ok := s.Metrics[metric]
	if !ok {
		return &e, errors.New("Invalid metric")
	}

	m := v.V2.GetLastValue()
	if m == nil {
		return m, errors.New("GetLastMetric: No metric found")
	}

	return m, nil
}

func (s *SqlServer) GetLastSqlPerSecond() int64 {
	m, err := s.GetLastMetric("sql")
	if err != nil {
		return 0
	}
	return m.ValuePerSecond
}

func (s *SqlServer) GetLastSqlPerSecondString() string {
	return humanize.Comma(s.GetLastSqlPerSecond())
}
