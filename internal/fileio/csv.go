package fileio

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ReadConfigCSV(name string) ([][]string, error) {
	result := make([][]string, 0)
	exe, err := os.Executable()
	if err != nil {
		return result, errors.Wrap(err, "os.executable")
	}
	wd := filepath.Dir(exe)
	fullfile := filepath.Join(wd, "config", name)

	if _, err := os.Stat(fullfile); err != nil {
		return result, errors.Wrap(err, "os.stat")
	}

	csvfile, err := os.Open(fullfile)
	if err != nil {
		return result, errors.Wrap(err, "os.open")
	}
	defer csvfile.Close()
	reader := csv.NewReader(csvfile)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("bad row in ag_names.csv: %+v", record)
		}
		result = append(result, record)
	}
	return result, nil
}
