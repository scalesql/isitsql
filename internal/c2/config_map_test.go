package c2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyFile(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{}
	_, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Zero(len(msgs))
}

func TestSimpleMap(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Instances: []Instance{
			{ID: "a"},
			{ID: "b"},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Zero(len(msgs))
	assert.Equal(2, len(fc.Connections))
}

func TestDuplicateKeys(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Instances: []Instance{
			{ID: "a"},
			{ID: "a"},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Equal(1, len(msgs))
	assert.Equal(1, len(fc.Connections))
}

func TestInvalidKey(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Instances: []Instance{
			{ID: "a$"},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Equal(1, len(msgs))
	assert.Equal(0, len(fc.Connections))
}

func TestDuplicateKeysUsingKey(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Instances: []Instance{
			{ID: "a"},
			{ID: "b", Key: ptr("a")},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Equal(1, len(msgs))
	assert.Equal(1, len(fc.Connections))
}

func TestFixedKeys(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Instances: []Instance{
			{ID: "D40\\SQL2016"},
			{ID: "a"},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Equal(0, len(msgs))
	assert.Equal(2, len(fc.Connections))
	assert.Contains(fc.Connections, "a")
	assert.Contains(fc.Connections, "d40-sql2016")
}

func TestMergedTags(t *testing.T) {
	assert := assert.New(t)
	cf := ConnectionFile{
		Defaults: &Defaults{
			Tags: ptr([]string{"base"}),
		},
		Instances: []Instance{
			{ID: "D40\\SQL2016", Tags: ptr([]string{"base"})},
			{ID: "a", Tags: ptr([]string{"New", "a"})},
		},
	}
	fc, msgs := makeMap([]string{"f1.hcl"}, []ConnectionFile{cf})
	assert.Equal(0, len(msgs))
	assert.Equal(2, len(fc.Connections))
	assert.Contains(fc.Connections, "a")
	//assert.Contains(fc.Connections, "D40-SQL2016")
	conn0, ok := fc.Connections["d40-sql2016"]
	assert.True(ok)
	assert.NotNil(conn0)
	assert.Equal([]string{"base"}, conn0.Tags)

	conn1, ok := fc.Connections["a"]
	assert.True(ok)
	assert.NotNil(conn1)
	assert.Equal([]string{"a", "base", "new"}, conn1.Tags)
}
