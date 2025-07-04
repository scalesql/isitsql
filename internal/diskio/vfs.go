package diskio

import (
	"database/sql"
	"time"
)

// VirtualFileStats holds the results of dm_io_virtual_file_stats
type VirtualFileStats struct {
	CaptureTime time.Time `json:"capture_time,omitempty"`
	SampleMS    int64     `json:"sample_ms,omitempty"`
	Reads       int64     `json:"reads,omitempty"`
	ReadBytes   int64     `json:"read_bytes,omitempty"`
	ReadStall   int64     `json:"read_stall,omitempty"`
	Writes      int64     `json:"writes,omitempty"`
	WriteBytes  int64     `json:"write_bytes,omitempty"`
	WriteStall  int64     `json:"write_stall,omitempty"`
	StallMS     int64     `json:"stall_ms,omitempty"`
}

// GetFileStats returns the file stats
func GetFileStats(db *sql.DB) (VirtualFileStats, error) {
	var s VirtualFileStats
	var err error

	row := db.QueryRow(`
        SELECT	
            MAX(sample_ms) AS [Milliseconds],
            SUM(num_of_reads) AS [Reads],
            SUM(num_of_bytes_read) AS [ReadBytes],
            SUM(io_stall_read_ms) AS [ReadStall],
            SUM(num_of_writes) AS [Writes],
            SUM(num_of_bytes_written) AS [WriteBytes],
            SUM(io_stall_write_ms) AS [WriteStall]
        FROM	sys.dm_io_virtual_file_stats(NULL, NULL) 

			`)

	err = row.Scan(&s.SampleMS, &s.Reads, &s.ReadBytes, &s.ReadStall, &s.Writes, &s.WriteBytes, &s.WriteStall)
	s.CaptureTime = time.Now()

	// If there aren't any rows just return nothing
	if err == sql.ErrNoRows {
		return s, sql.ErrNoRows
	}

	if err != nil {
		return s, err
	}

	return s, err
}

// Add adds the file stats to another entry
func (b *VirtualFileStats) Add(a VirtualFileStats) {
	//var s VirtualFileStats
	seconds := a.SampleMS / 1000
	if seconds > 0 {
		b.Reads += a.Reads / seconds
		b.Writes += a.Writes / seconds
		b.ReadBytes += a.ReadBytes / seconds
		b.WriteBytes += a.WriteBytes / seconds
	}
}

// Sub returns the difference in disk IO.  Reset
func (b *VirtualFileStats) Sub(a VirtualFileStats, reset bool) VirtualFileStats {
	var s VirtualFileStats

	// if the before value is older or we are resetting stats,
	// just leave the delta values at zero
	if a.SampleMS >= b.SampleMS || reset {
		return s
	}

	s.CaptureTime = b.CaptureTime
	s.SampleMS = b.SampleMS - a.SampleMS
	s.Reads = b.Reads - a.Reads
	s.ReadBytes = b.ReadBytes - a.ReadBytes
	s.ReadStall = b.ReadStall - a.ReadStall
	s.Writes = b.Writes - a.Writes
	s.WriteBytes = b.WriteBytes - a.WriteBytes
	s.WriteStall = b.WriteStall - a.WriteStall

	// This is a hack for snapshots
	// When they get created, their new values cause everything to go negative
	// Need to track per database and only handle increases
	// Yuck
	if s.Reads < 0 || s.ReadBytes < 0 || s.Writes < 0 || s.WriteBytes < 0 {
		s.Reads = 0
		s.ReadBytes = 0
		s.Writes = 0
		s.WriteBytes = 0
	}

	return s
}
