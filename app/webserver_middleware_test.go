package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopFolder(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		got  string
		want string
	}
	tests := []test{
		{"", ""},
		{"/", ""},
		{"/about", "about"},
		{"//about", "about"},
		{"/about/values", "about"},
		{"/metrics/isitsql", "metrics"},
		{"/api/cpu/d40-sql2022", "api"},
		{"/static/js/isitsql-charting-1.0.js", "static"},
	}
	for _, tc := range tests {
		top := topfolder(tc.got)
		assert.Equal(tc.want, top)
	}
}
