package app

import (
	"encoding/xml"
	"strconv"
	"time"
)

type sqlRingBufferType struct {
	Name string `xml:"name,attr"`
}

type sqlRingBufferData struct {
	Name  string            `xml:"name,attr"`
	Value string            `xml:"value"`
	Type  sqlRingBufferType `xml:"type"`
}

type sqlRingBufferAction struct {
	Name  string            `xml:"name,attr"`
	Value string            `xml:"value"`
	Type  sqlRingBufferType `xml:"type"`
}

type sqlRingBufferEvent struct {
	Name         string                `xml:"name,attr"`
	TimeStamp    time.Time             `xml:"timestamp,attr"`
	DataValues   []sqlRingBufferData   `xml:"data"`
	ActionValues []sqlRingBufferAction `xml:"action"`
}

type sqlRingBuffer struct {
	XMLName    xml.Name             `xml:"RingBufferTarget"`
	EventCount int                  `xml:"eventCount,attr"`
	Events     []sqlRingBufferEvent `xml:"event"`
}

type xeSession struct {
	EventSessionAddress []byte
	Name                string
	TargetData          string
}

type xEvent struct {
	EventName   string
	ErrorNumber int
	TimeStamp   time.Time
	UTCTime     time.Time
	Message     string
	Severity    int
	State       int
	HostName    string
	AppName     string
	UserName    string
	SQLText     string
	DatabaseID  int
}

func (s *SqlServerWrapper) getXESessions() ([]*xEvent, error) {

	var err error

	stmt := `
		SELECT -- TOP 1 
				[event_session_address], [name], [target_data] 
		FROM 	sys.dm_xe_session_targets st 
		JOIN 	sys.dm_xe_sessions s ON s.address = st.event_session_address 
		WHERE 	1=1
		AND 	(name like 'isitsql%' OR name in ('ErrorSession','PermissionSession'))
		AND target_name = 'ring_buffer'; 
`
	s.RLock()
	db := s.DB
	s.RUnlock()
	rows, err := db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//var sessions []*xeSession
	var events []*xEvent
	var sqlErrorNum int

	for rows.Next() {
		xe := new(xeSession)
		err := rows.Scan(&xe.EventSessionAddress, &xe.Name, &xe.TargetData)
		if err != nil {
			return nil, err
		}

		//sessions = append(sessions, xe)

		v := sqlRingBuffer{EventCount: -2}

		err = xml.Unmarshal([]byte(xe.TargetData), &v)
		if err != nil {
			return nil, err
		}

		// Get the values from the event
		for _, y := range v.Events {
			var e xEvent
			e.EventName = y.Name
			e.TimeStamp = y.TimeStamp

			for _, d := range y.DataValues {
				switch d.Name {

				// SQL Server 2012 and higher use this for the error number
				case "error_number":
					sqlErrorNum, _ = strconv.Atoi(d.Value)
					if sqlErrorNum != 0 {
						e.ErrorNumber = sqlErrorNum
					}

				// SQL Server 2008 uses this for the error number
				case "error":
					sqlErrorNum, _ = strconv.Atoi(d.Value)
					if sqlErrorNum != 0 {
						e.ErrorNumber = sqlErrorNum
					}

				case "severity":
					e.Severity, _ = strconv.Atoi(d.Value)

				case "state":
					e.State, _ = strconv.Atoi(d.Value)

				case "message":
					e.Message = d.Value

				}
			}

			for _, a := range y.ActionValues {
				switch a.Name {

				case "sql_text":
					e.SQLText = a.Value

				case "client_hostname":
					e.HostName = a.Value

				case "client_app_name":
					e.AppName = a.Value

				case "username":
					e.UserName = a.Value

				case "database_id":
					e.DatabaseID, _ = strconv.Atoi(a.Value)
				}
			}

			events = append(events, &e)
		}
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return events, nil
}

// type SQLRingBufferType struct {
// 	Name string `xml:"name,attr"`
// }

// type SQLRingBufferData struct {
// 	Name string `xml:"name,attr"`
// 	//Type string `xml:"type>name,attr"`
// 	Value string `xml:"value"`
// 	Type SQLRingBufferType `xml:"type"`
// }

// type SQLRingBufferAction struct {
// 	Name string `xml:"name,attr"`
// 	//Type string `xml:"type>name,attr"`
// 	Value string `xml:"value"`
// 	Type SQLRingBufferType `xml:"type"`
// }

// type SQLRingBufferEvent struct {
// 	Name       string              `xml:"name,attr"`
// 	TimeStamp  time.Time           `xml:"timestamp,attr"`
// 	DataValues []SQLRingBufferData `xml:"data"`
// 	ActionValues []SQLRingBufferAction `xml:"action"`

// }

// type SQLRingBuffer struct {
// 	XMLName    xml.Name             `xml:"RingBufferTarget"`
// 	EventCount int                  `xml:"eventCount,attr"`
// 	Events     []SQLRingBufferEvent `xml:"event"`
// }

// v := SQLRingBuffer{EventCount: -2}

// err := xml.Unmarshal([]byte(data), &v)
// if err != nil {
// 	fmt.Printf("error: %v", err)
// 	return
// }
// fmt.Printf("XMLName: %#v\n", v.XMLName)
// fmt.Printf("eventCount: %#v\n", v.EventCount)
// fmt.Printf("v: %#v\n", v.Events)

// for _, y := range v.Events {
// 	fmt.Printf("y: %#v\n", y)
// 	for _, d := range y.DataValues {
// 		fmt.Printf("    d: %#v\n", d)
// 	}
// 	fmt.Printf("\n")
// 	for _, a := range y.ActionValues {
// 		fmt.Printf("    a: %#v\n", a)
// 	}
// 	fmt.Printf("\n")
// }
