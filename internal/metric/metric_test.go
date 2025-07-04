package metric

import (
	"testing"
	"time"
)

// func TestInit(t *testing.T) {
// 	k := Key{"test", CPU}
// 	//c := Capacity(k)
// 	if c != 0 {
// 		t.Error("Expected 0, got ", c)
// 	}
// }

func TestReset(t *testing.T) {
	Reset()
}

func TestEnqueueFirst(t *testing.T) {
	key := Key{"test1", CPU}
	val := Value{
		TimeStamp: time.Now(),
		Value:     37,
		Delta:     0,
	}
	err := Enqueue(key, val)
	if err != nil {
		t.Error("Error: ", err)
	}
}

func TestEnqueueFew(t *testing.T) {
	key := Key{"test2", CPU}
	val := Value{
		TimeStamp: time.Now(),
		Value:     37,
		Delta:     0,
	}
	err := Enqueue(key, val)
	if err != nil {
		t.Error("First Error: ", err)
	}

	v2 := Value{TimeStamp: time.Now(), Value: 38}
	err = Enqueue(key, v2)
	if err != nil {
		t.Error("Second Error: ", err)
	}
}

func TestEnqueue200(t *testing.T) {
	k := Key{"test3", CPU}
	for i := 0; i < 200; i++ {
		v := Value{
			TimeStamp: time.Now(),
			Value:     int64(i),
		}
		err := Enqueue(k, v)
		if err != nil {
			t.Error("Error: ", i, err)
		}

	}
}

// Test is failing - research later
// func TestValues(t *testing.T) {
// 	Reset()
// 	var err error
// 	err = enqueue("test3", CPU, 7)
// 	if err != nil {
// 		t.Error("enqueue: ", err)
// 	}
// 	j, err := Values(Key{ServerKey: "test3", Type: CPU})
// 	if err != nil {
// 		t.Error("Values: ", err)
// 	}
// 	fmt.Println(j)
// }

func enqueue(uid string, t Type, val int64) error {
	k := Key{uid, t}
	v := Value{TimeStamp: time.Now(), Value: val}
	err := Enqueue(k, v)
	if err != nil {
		return err
	}
	return nil

}
