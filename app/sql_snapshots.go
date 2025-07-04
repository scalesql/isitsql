package app

import (
	"time"

	"github.com/pkg/errors"
)

type Snapshot struct {
	Name       string
	Source     string
	Created    time.Time
	CreatedUTC time.Time
	Size       int64
}

func (s *SqlServerWrapper) getSnapshots() error {
	var err error
	snaps := make([]Snapshot, 0)

	dbQuery := `
		;WITH  CTE AS (
			SELECT dbid as database_id, SUM(BytesOnDisk) as bytes_on_disk
			FROM fn_virtualfilestats(NULL,NULL) 
			group by dbid)
		SELECT	snap.[name]
			,src.[name]
			,snap.create_date
			,DATEADD(second, DATEDIFF(second, GETDATE(), GETUTCDATE()), snap.create_date) as create_date_utc
			,COALESCE(bytes_on_disk, 0) AS bytes_on_disk
		FROM sys.databases snap
		JOIN sys.databases src on src.database_id = snap.source_database_id
		LEFT JOIN CTE on CTE.database_id = snap.database_id
		WHERE snap.source_database_id IS NOT NULL
		ORDER BY snap.[name];
	`
	rows, err := s.DB.Query(dbQuery)
	if err != nil {
		return errors.Wrap(err, "query")
	}
	defer rows.Close()

	for rows.Next() {
		var snap Snapshot
		err = rows.Scan(&snap.Name, &snap.Source, &snap.Created, &snap.CreatedUTC, &snap.Size)
		if err != nil {
			return errors.Wrap(err, "rows.scan")
		}
		snaps = append(snaps, snap)
	}
	if err = rows.Err(); err != nil {
		return errors.Wrap(err, "rows.err")
	}
	s.Lock()
	s.Snapshots = snaps
	s.Unlock()
	return nil
}
