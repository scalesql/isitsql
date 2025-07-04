package app

import (
	"encoding/xml"
)

// 	testxml := `
// <Record id="2703" type="RING_BUFFER_SCHEDULER_MONITOR" time="544339365">
//   <SchedulerMonitorEvent>
//     <SystemHealth>
//       <ProcessUtilization>2</ProcessUtilization>
//       <SystemIdle>98</SystemIdle>
//       <UserModeTime>13750000</UserModeTime>
//       <KernelModeTime>10312500</KernelModeTime>
//       <PageFaults>491</PageFaults>
//       <WorkingSetDelta>827392</WorkingSetDelta>
//       <MemoryUtilization>37</MemoryUtilization>
//     </SystemHealth>
//   </SchedulerMonitorEvent>
// </Record>
// `

//sh, err := GetCpu(testxml)
//  if err != nil {
// 	 log.Println(err)
//  }
//
//  log.Println(sh.XMLName)
//  log.Printf("CPU: %d", sh.ID)
//  log.Println(sh)
//  log.Println(sh.SME[0].SH[0].ProcessUtilization)

type SystemHealthRecord struct {
	XMLName xml.Name                           `xml:"Record"`
	ID      int                                `xml:"id,attr"`
	SME     []SystemHealthScheduleMonitorEvent `xml:"SchedulerMonitorEvent"`
}

type SystemHealthScheduleMonitorEvent struct {
	XMLName xml.Name          `xml:"SchedulerMonitorEvent"`
	SH      []SystemHealthXml `xml:"SystemHealth"`
}

type SystemHealthXml struct {
	XMLName            xml.Name `xml:"SystemHealth"`
	ProcessUtilization int      `xml:"ProcessUtilization"`
	SystemIdle         int      `xml:"SystemIdle"`
}

func GetCpu(x string) (SystemHealthRecord, error) {
	var r SystemHealthRecord
	byteArray := []byte(x)
	err := xml.Unmarshal(byteArray, &r)
	if err != nil {
		return r, err
	}
	return r, nil
}
