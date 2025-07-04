package bucket

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurge(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	fs := afero.NewMemMapFs()
	clk := clock.NewMock()
	clk.Set(time.Date(2009, time.November, 10, 23, 01, 20, 30, time.UTC))

	// write an extra file with a wrong extension and prefix
	fileName := filepath.Join(".", "junk", fmt.Sprintf("%s_%s.%s", "what", clk.Now().UTC().Format("20060102_150405"), "nope"))
	err := afero.WriteFile(fs, fileName, []byte("test"), 0644)
	require.NoError(err)

	// write four log files
	for i := 0; i <= 3; i++ {
		fileName := filepath.Join(".", "junk", fmt.Sprintf("%s_%s.%s", "myprefix", clk.Now().UTC().Format("20060102_150405"), "zzz"))
		err := afero.WriteFile(fs, fileName, []byte("test"), 0644)
		require.NoError(err)
		clk.Add(48 * time.Hour)
	}
	f1, err := afero.ReadDir(fs, filepath.Join(".", "junk"))
	require.NoError(err)
	assert.Len(f1, 5)

	//println("now:      ", clk.Now().Format(time.RFC3339))
	err = purgeFilesPath(fs, clk, filepath.Join(".", "junk"), "myprefix", "zzz", 24*5*time.Hour)
	assert.NoError(err)

	f2, err := afero.ReadDir(fs, filepath.Join(".", "junk"))
	require.NoError(err)
	assert.Len(f2, 3)
}
