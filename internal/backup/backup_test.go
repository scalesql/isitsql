package backup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetFull(t *testing.T) {
	SetFull("domain1", "ag1", "db", time.Now(), "I1", "D1")
	SetFull("domain1", "ag1", "db", time.Now(), "I1", "D1")
	SetFull("domain1", "ag1", "db", time.Now(), "I1", "D1")
	SetFull("domain1", "ag1", "db", time.Now(), "I1", "D1")
}

func TestGet(t *testing.T) {

	SetFull("domain1", "ag2", "db1", time.Now(), "I1", "D1")

	b, found := Get("domain1", "AG2", "DB1")
	if !found {
		t.Error("backup not found")
	}
	if b.FullInstance != "I1" {
		t.Errorf("wrong instance.  Expected %s got %s", "I1", b.FullInstance)
	}

	SetFull("domain1", "ag2", "DB1", time.Now().Add(1*time.Second), "I2", "D2")

	b2, found := Get("domain1", "AG2", "db1")
	if !found {
		t.Error("backup not found")
	}
	if b2.FullInstance != "I2" {
		t.Errorf("wrong instance.  Expected %s got %s", "I1", b2.FullInstance)
	}
	if b2.FullDevice != "D2" {
		t.Errorf("Wrong Full Device.  Expected %s got %s", "D2", b2.FullDevice)
	}
}

func TestDomain(t *testing.T) {
	assert := assert.New(t)
	date1 := time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC)
	SetFull("domain1", "ag1", "db1", date1, "I1", "D1")
	SetFull("domain2", "ag1", "db1", date2, "I1", "D1")

	b1, ok := Get("domain1", "ag1", "db1")
	assert.True(ok)
	assert.Equal(date1, b1.FullStarted)

	b2, ok := Get("domain2", "ag1", "db1")
	assert.True(ok)
	assert.Equal(date2, b2.FullStarted)
}

func TestDelete(t *testing.T) {
	SetFull("domain1", "ag3", "db5", time.Now(), "I1", "D1")
	SetFull("domain1", "ag3", "db5", time.Now(), "I1", "D1")
	Delete("domain1", "AG3", "db5")
	_, f1 := Get("domain1", "ag3", "DB5")
	if f1 {
		t.Error("Delete failed.  Backup shouldn't exist")
	}
	_, f2 := Get("domain1", "ag3", "db5")
	if f2 {
		t.Error("Delete failed.  Backup shouldn't exist")
	}
}

func TestTypes(t *testing.T) {
	var ts time.Time
	first := time.Now()
	second := time.Now().Add(12 * time.Hour)

	SetFull("domain1", "ag5", "db1", first, "I1", "D1")
	b1, found := Get("domain1", "ag5", "db1")
	if !found {
		t.Error("Can't get backup")
	}
	if b1.LogInstance != "" || b1.LogStarted != ts {
		t.Error("Log backups aren't empty", b1)
	}

	SetLog("domain1", "ag5", "db1", second, "I2", "L2")
	b2, found := Get("domain1", "ag5", "db1")
	if !found {
		t.Error("Can't get backup")
	}
	if b2.LogInstance != "I2" || b2.LogStarted != second || b2.FullInstance != "I1" || b2.FullStarted != first || b2.LogDevice != "L2" {
		t.Error("Bad values: ", b2)
	}
}

func TestLog(t *testing.T) {
	var ts time.Time
	first := time.Now()
	second := time.Now().Add(12 * time.Hour)

	SetLog("domain1", "ag5", "db5", second, "I3", "L1")
	SetLog("domain1", "ag5", "db5", first, "bad", "L1")

	b2, found := Get("domain1", "ag5", "db5")
	if !found {
		t.Error("Can't get backup")
	}
	if b2.LogInstance != "I3" || b2.LogStarted != second || b2.FullInstance != "" || b2.FullStarted != ts {
		t.Error("Bad values: ", b2)
	}

}

func TestSet(t *testing.T) {
	var zero time.Time
	Set("domain1", "ag6", "db1")
	Set("domain1", "ag6", "db2")
	Set("domain1", "ag6", "db3")
	b, found := Get("domain1", "ag6", "db2")
	if !found {
		t.Error("Set didn't set")
	}
	if b.FullInstance != "" || b.FullStarted != zero || b.LogInstance != "" || b.LogStarted != zero {
		t.Error("TestSet should get empty rows", b)
	}
}
