package app

import (
	"time"

	"github.com/scalesql/isitsql/internal/cpuring"
	"github.com/pkg/errors"
)

func (s *SqlServerWrapper) GetCPU2() error {
	s.RLock()
	db := s.DB
	incontainer := s.InContainer
	s.RUnlock()

	// we convert the time to UTC and then a standard string
	// otherwise the driver defaults to the current timezone instead of UTC
	stmt := `
	SET NOCOUNT ON;
	declare @ts_now bigint 
select @ts_now = ms_ticks from sys.dm_os_sys_info 
			
select	event_utc_string = CONVERT(NVARCHAR(100), 
								DATEADD(second, 
										DATEDIFF(second, GETDATE(), GETUTCDATE()), dateadd (ms, (y.[timestamp] -@ts_now), 
										GETDATE())), 
								126) + 'Z'
		,ticks = timestamp
		,SQLProcessUtilization 
		,100 - SystemIdle - SQLProcessUtilization as OtherProcessUtilization 
from ( 
	select 
	record.value('(./Record/@id)[1]', 'int') as record_id, 
	record.value('(./Record/SchedulerMonitorEvent/SystemHealth/SystemIdle)[1]', 'int') 
	as SystemIdle, 
	record.value('(./Record/SchedulerMonitorEvent/SystemHealth/ProcessUtilization)[1]', 
	'int') as SQLProcessUtilization, 
	[timestamp] 
	from ( 
		select TOP 60 timestamp, convert(xml, record) as record 
		from sys.dm_os_ring_buffers 
		where ring_buffer_type = N'RING_BUFFER_SCHEDULER_MONITOR' 
		and record like '%<SystemHealth>%'
		ORDER BY [timestamp] DESC
		) as x 
) as y 
where dateadd (ms, (y.[timestamp] -@ts_now), GETDATE()) > DATEADD(MINUTE, -65, GETDATE())
order by [timestamp] asc


	`

	rows, err := db.Query(stmt)
	if err != nil {
		return errors.Wrap(err, "db.query")
	}

	defer rows.Close()
	var lastCPU, lastSQLCPU int
	cpuevents := make([]cpuring.CPU, 0, 60)
	for rows.Next() {
		var result struct {
			utcString string
			ticks     int64
			sql       int
			other     int
		}
		err = rows.Scan(&result.utcString, &result.ticks, &result.sql, &result.other)
		if err != nil {
			return errors.Wrap(err, "scan")
		}
		// if in a container, don't record the "other" CPU
		if incontainer {
			result.other = 0
		}
		// Parse the UTC string and conver to local time
		utc, err := time.Parse(time.RFC3339Nano, result.utcString)
		if err != nil {
			return errors.Wrapf(err, "cpu: invalid time: %s", result.utcString)
		}
		utc = utc.UTC()
		// localTime := utc.Local()
		// fmt.Printf("utc: %s     local: %s\n", utc, localTime)

		newcpu := cpuring.CPU{
			At:    utc.Local(),
			SQL:   result.sql,
			Other: result.other}
		cpuevents = append(cpuevents, newcpu)

		lastCPU = result.sql + result.other
		lastSQLCPU = result.sql
	}

	s.Lock()
	defer s.Unlock()

	// Populate the new values
	lastcpu, found := s.CPUUsage.GetNewest()
	for _, cpu := range cpuevents {
		// converting from ticks to a time has jitter but it always seems to be less than 1s
		if !found || cpu.At.Sub(lastcpu.At) > 1*time.Second {
			newcpu := cpu
			s.CPUUsage.Enqueue(&newcpu)
			//fmt.Printf("%s:   adding: %s\n", s.MapKey, cpu.String())
		}
	}

	s.LastPollTime = time.Now()
	s.LastCpu = lastCPU
	s.LastSQLCPU = lastSQLCPU
	s.CoresUsedSQL = float32(s.LastSQLCPU) * float32(s.CpuCount) / 100
	s.CoresUsedOther = float32(s.LastCpu-s.LastSQLCPU) * float32(s.CpuCount) / 100

	return nil
}
