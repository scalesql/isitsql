package backup

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/scalesql/isitsql/internal/files"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetIgnoredBackups into a string array
func GetIgnoredBackups() ([][]string, error) {
	f, err := files.GetFirst("config/ignoredBackups.csv", "config/ignoredBackups.txt")
	if os.IsNotExist(err) {
		return nil, nil
	}

	/* #nosec G304 */
	csvfile, err := os.Open(f)
	if err != nil {
		return nil, errors.Wrap(err, "os.open")
	}

	defer func() {
		if err := csvfile.Close(); err != nil {
			logrus.Error(errors.Wrap(err, "csvfile.close"))
		}
	}()

	reader := csv.NewReader(csvfile)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	// Set to -1 to allow a variable number of fields
	reader.FieldsPerRecord = -1

	ignored, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// trim all the fields
	for i, line := range ignored {
		for j := range line {
			ignored[i][j] = strings.TrimSpace(ignored[i][j])
		}
	}

	return ignored, nil
}
