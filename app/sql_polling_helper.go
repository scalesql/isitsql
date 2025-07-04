package app

import (
	"fmt"
	"math"

	"github.com/dustin/go-humanize"

	"regexp"
)

func (as ActiveSession) StartTimeString() string {
	return humanize.Time(as.StartTime)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

// KBToString returns a string of a KB
// Replace this with a generic function and not a method
func KBToString(b int64) string {
	ui := uint64(b) * 1024
	return BytesToString(ui)
}

// Int64ToPct returns a/b * 100
func Int64ToPct(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	fa := float64(a)
	fb := float64(b)
	return (fa / fb) * 100
}

func BytesToString(s uint64) string {
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1024, sizes)
}

func (s *SqlServer) LastPollTimeString() string {
	if s.LastPollTime.IsZero() {
		return "never"
	}

	return humanize.Time(s.LastPollTime)

}

func (s *SqlServer) UpTimeString() string {
	if s.StartTime.IsZero() || s.CurrentTime.IsZero() {
		return "uknown"
	}
	return durationToShortString(s.StartTime, s.CurrentTime)
}

// KBToString returns a string of a KB
// Replace this with a generic function and not a method
func (s *SqlServer) KBToString(b int64) string {
	return KBToString(b)
}

// KBToString returns a string of a KB
func (d Database) KBToString(b int64) string {
	return KBToString(b)
}

// MemoryPercent returns the percent of memory used * 100
func (s *SqlServer) MemoryPercent() int64 {

	if s.PhysicalMemoryKB == 0 {
		return 0
	}
	return (s.SqlServerMemoryKB * 100) / s.MemoryCap()

}

// MaxMemorySet determines if the maximum memory is set
func (s *SqlServer) MaxMemorySet() bool {
	// if s.MaxMemoryKB == 2147483647*1024 {
	// 	return false
	// }
	// return true
	return s.MaxMemoryKB != 2147483647*1024
}

// MemoryCap returns the lower of physical memory or max memory
func (s *SqlServer) MemoryCap() int64 {
	if s.MaxMemoryKB < s.PhysicalMemoryKB {
		return s.MaxMemoryKB
	}
	return s.PhysicalMemoryKB
}

// GetTableCssClass determines if the row should be red to alert
func (s *SqlServer) GetTableCssClass() string {
	if s.LastPollError != "" {
		return "alert alert-danger"
	}
	return ""

}

// PeakCoresUsed returns the peak cores for SQL usage and other usage over the last hour
func (s *SqlServer) PeakCoresUsed() (peakCoresSQL float32, peakCoresOther float32) {
	usage := s.CPUUsage.Values()
	for _, u := range usage {
		usql := float32(u.SQL) * float32(s.CpuCount) / 100
		if usql > peakCoresSQL {
			peakCoresSQL = usql
		}
		uother := float32(u.Other) * float32(s.CpuCount) / 100
		if uother > peakCoresOther {
			peakCoresOther = uother
		}
	}
	return
}

// AverageCoresUsed returns the average cores for SQL usage and other usage over the last hour
func (s *SqlServer) AverageCoresUsed() (avgCoresSQL float32, avgCoresOther float32) {
	usage := s.CPUUsage.Values()
	var totSQL float32
	var totOther float32
	var count float32
	for _, u := range usage {
		totSQL += float32(u.SQL) * float32(s.CpuCount) / 100
		totOther += float32(u.Other) * float32(s.CpuCount) / 100
		count++
	}
	if count > 0 {
		avgCoresSQL = totSQL / count
		avgCoresOther = totOther / count
	}
	return
}

// ProductVersionString returns SQL Server 2016
// It expects 11.0.3453.1
func (s *SqlServer) ProductVersionString(v string) string {
	// s.RLock()
	// defer s.RUnlock()

	if v == "" {
		return "SQL Server Unknown"
	}

	if v[0:3] == "9.0" {
		return "SQL Server 2005"
	}

	if v[0:3] == "8.0" {
		return "SQL Server 2000"
	}

	if len(v) < 4 {
		return "SQL Server " + v
	}

	if v[0:4] == "16.0" {
		return "SQL Server 2022"
	}

	if v[0:4] == "15.0" {
		return "SQL Server 2019"
	}

	if v[0:4] == "14.0" {
		return "SQL Server 2017"
	}

	if v[0:4] == "13.0" {
		return "SQL Server 2016"
	}

	if v[0:4] == "12.0" {
		return "SQL Server 2014"
	}

	if v[0:4] == "11.0" {
		return "SQL Server 2012"
	}

	if v[0:4] == "10.5" {
		return "SQL Server 2008 R2"
	}

	if v[0:4] == "10.0" {
		return "SQL Server 2008"
	}

	return "SQL Server " + v[0:4]
}

// LastPollErrorClean fixes up the text error string and limits it to 45 characters
func (s *SqlServer) LastPollErrorClean(length int) string {
	// s.RLock()
	// defer s.RUnlock()

	r, err := regexp.Compile(`\[.*?\]`)
	if err != nil {
		return s.LastPollError
	}

	e := r.ReplaceAllString(s.LastPollError, "")

	if len(e) > 18 {
		// fmt.Println(e[0:18])
		if e[0:18] == "SQLDriverConnect: " {
			e = e[18:]
		}
	}

	l := len(e)

	if l > length {
		return e[0:length] + "..."
	}
	return e
}
