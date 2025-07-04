package bucket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type FakeEvent struct {
	Wait  string
	Value int
}

func TestBucket(t *testing.T) {
	assert := assert.New(t)

	var err error
	bw := BucketWriter{}
	bw.fs = afero.NewMemMapFs()
	myclock := clock.NewMock()
	bw.clock = myclock
	// fmt.Printf("%+v\n", bw)
	bw.Start(".", "test")
	// fmt.Printf("%+v\n", bw)
	err = bw.rollover()
	assert.NoError(err)
	// fmt.Printf("%+v\n", bw)
	files, err := afero.ReadDir(bw.fs, bw.path)
	assert.NoError(err)
	assert.Equal(1, len(files))

	// rollover no time
	err = bw.rollover()
	assert.NoError(err)
	files, err = afero.ReadDir(bw.fs, bw.path)
	assert.NoError(err)
	assert.Equal(1, len(files))

	// rollover after 15 minutes
	println(bw.clock.Now().Format(time.RFC3339))
	myclock.Add(15 * time.Minute)
	err = bw.rollover()
	assert.NoError(err)
	files, err = afero.ReadDir(bw.fs, bw.path)
	assert.NoError(err)
	assert.Equal(2, len(files))
	println(bw.clock.Now().Format(time.RFC3339))

	fake := FakeEvent{"DISK", 37}
	err = bw.Write("abc", fake)
	assert.NoError(err)

	f := *bw.file
	name := f.Name()
	myclock.Add(15 * time.Minute)
	err = bw.rollover()
	assert.NoError(err)

	written, err := afero.ReadFile(bw.fs, name)
	assert.NoError(err)
	println(string(written))
	var se2 ServerEvent
	err = json.Unmarshal(written, &se2)
	assert.NoError(err)
	assert.Equal("abc", se2.MapKey)

	var fake2 FakeEvent
	err = json.Unmarshal([]byte(se2.Payload), &fake2)
	assert.NoError(err)
	assert.Equal(fake.Wait, fake2.Wait)
	assert.Equal(fake.Value, fake2.Value)
}
