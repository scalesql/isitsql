package gui

import (
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/mssql"
	"github.com/scalesql/isitsql/internal/mssql/session"
	"github.com/dustin/go-humanize"
)

// TemplateFuncs is used by the HTML templates for
// various things.  It is imported into templates.
var TemplateFuncs = template.FuncMap{
	// "isBeta": func() bool {
	// 	//defer recovery()
	// 	c := getGlobalConfig()
	// 	return c.IsBeta
	// },
	// "isEnterprise": func() bool {
	// 	c := getGlobalConfig()
	// 	return c.IsEnterprise
	// },
	"kbtostring": func(b int64) string {
		return KBToString(b)
	},
	"kbint2string": func(b int) string {
		return KBToString(int64(b))
	},
	"int64topct": func(a, b int64) float64 {
		return Int64ToPct(a, b)
	},
	"utctolocal": func(t time.Time) time.Time {
		loc, _ := time.LoadLocation("Local")
		return t.In(loc)
	},
	"localToUTC": func(t time.Time) time.Time {
		return t.UTC()
	},
	"xeSessionTime": func(t time.Time) string {
		tz, _ := time.LoadLocation("Local")
		local := t.In(tz)
		return local.Format("2006-01-02 15:04:05")
	},
	"timetoYMDT": func(t time.Time) string {
		return t.Format("2006-01-02 3:04:05 PM")
	},
	"timetoYMD": func(t time.Time) string {
		return t.Format("2006-01-02")
	},

	"leftstring200": func(s string) string {
		s2 := strings.Replace(s, ",", ", ", -1)
		if len(s2) <= 200 {
			return s2
		}
		return session.TrimSQL(s2, 200) + "..."
	},
	"comma": func(b int64) string {
		return humanize.Comma(b)
	},
	"commaint": func(b int) string {
		return humanize.Comma(int64(b))
	},
	"shortDuration": func(a time.Time) string {
		return durationToShortString(a, time.Now())
	},
	"durationDifference": func(a, b time.Time) string {
		return durationToShortString(a, b)
	},
	"mstoshortstring": func(s int) string {
		return millisecondsToShortString(s)
	},
	"divide": func(a, b int64) int64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"divideint": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"bytes": func(s int64) string {
		r := uint64(s)
		sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
		return humanateBytes(r, 1024, sizes)
	},
	"bytesint": func(s int) string {
		r := uint64(s)
		sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
		return humanateBytes(r, 1024, sizes)
	},
	"hasTimeValue": func(t time.Time) bool {
		var zero time.Time
		// if t == zero {
		// 	return false
		// }
		// return true
		return t != zero
	},
	"arrayToCSV": func(a []string) string {
		return strings.Join(a, ", ")
	},
	"now": func() time.Time {
		return time.Now()
	},
	"versiontostring": func(v string) string {
		return mssql.VersionToString(v)
	},
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

func durationToShortString(a, b time.Time) string {
	if a.IsZero() {
		return "never"
	}

	if a.After(b) {
		z := a.Sub(b)
		if z.Minutes() < 5 {
			return "now"
		}
		return "invalid"
		// fmt.Println(a)
		// fmt.Println(b)
	}

	up := b.Sub(a)

	// Days
	if up.Hours() > 72 {
		h := int(up.Hours())
		d := h / 24
		s := fmt.Sprintf("%dd", d)
		return s
	}

	if up.Minutes() > 99 {
		m := int(up.Minutes())
		h := m / 60
		s := fmt.Sprintf("%dh", h)
		return s
	}

	if up.Seconds() > 99 {
		s := int(up.Seconds())
		m := s / 60
		return fmt.Sprintf("%dm", m)
	}

	if up.Seconds() < 1 {
		return "now"
	}

	return fmt.Sprintf("%ds", int(up.Seconds()))
}

func millisecondsToShortString(s int) string {
	if s < 1000 {
		return fmt.Sprintf("%dms", s)
	}
	return SecondsToShortString(s / 1000)
}
