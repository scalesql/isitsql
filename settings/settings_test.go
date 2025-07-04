package settings

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/pkg/errors"
)

func TestMain(m *testing.M) {
	code := m.Run()
	_ = printJSON()
	os.Exit(code)
}

func printJSON() error {
	s, err := ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnetions")
	}

	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return errors.Wrap(err, "marshall")
	}

	log.Println("===============================================")
	log.Println(string(b))
	log.Println("===============================================")

	return nil
}
