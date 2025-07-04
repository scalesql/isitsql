package backup

import (
	"testing"
	"time"
)

func TestSetFull(t *testing.T) {
	SetFull("ag1", "db", time.Now(), "I1", "D1")
	SetFull("ag1", "db", time.Now(), "I1", "D1")
	SetFull("ag1", "db", time.Now(), "I1", "D1")
	SetFull("ag1", "db", time.Now(), "I1", "D1")
}

func TestGet(t *testing.T) {

	SetFull("ag2", "db1", time.Now(), "I1", "D1")

	b, found := Get("AG2", "DB1")
	if !found {
		t.Error("backup not found")
	}
	if b.FullInstance != "I1" {
		t.Errorf("wrong instance.  Expected %s got %s", "I1", b.FullInstance)
	}

	SetFull("ag2", "DB1", time.Now().Add(1*time.Second), "I2", "D2")

	b2, found := Get("AG2", "db1")
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

func TestDelete(t *testing.T) {
	SetFull("ag3", "db5", time.Now(), "I1", "D1")
	SetFull("ag3", "db5", time.Now(), "I1", "D1")
	Delete("AG3", "db5")
	_, f1 := Get("ag3", "DB5")
	if f1 {
		t.Error("Delete failed.  Backup shouldn't exist")
	}
	_, f2 := Get("ag3", "db5")
	if f2 {
		t.Error("Delete failed.  Backup shouldn't exist")
	}
}

func TestTypes(t *testing.T) {
	var ts time.Time
	first := time.Now()
	second := time.Now().Add(12 * time.Hour)

	SetFull("ag5", "db1", first, "I1", "D1")
	b1, found := Get("ag5", "db1")
	if !found {
		t.Error("Can't get backup")
	}
	if b1.LogInstance != "" || b1.LogStarted != ts {
		t.Error("Log backups aren't empty", b1)
	}

	SetLog("ag5", "db1", second, "I2", "L2")
	b2, found := Get("ag5", "db1")
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

	SetLog("ag5", "db5", second, "I3", "L1")
	SetLog("ag5", "db5", first, "bad", "L1")

	b2, found := Get("ag5", "db5")
	if !found {
		t.Error("Can't get backup")
	}
	if b2.LogInstance != "I3" || b2.LogStarted != second || b2.FullInstance != "" || b2.FullStarted != ts {
		t.Error("Bad values: ", b2)
	}

}

func TestSet(t *testing.T) {
	var zero time.Time
	Set("ag6", "db1")
	Set("ag6", "db2")
	Set("ag6", "db3")
	b, found := Get("ag6", "db2")
	if !found {
		t.Error("Set didn't set")
	}
	if b.FullInstance != "" || b.FullStarted != zero || b.LogInstance != "" || b.LogStarted != zero {
		t.Error("TestSet should get empty rows", b)
	}
}
