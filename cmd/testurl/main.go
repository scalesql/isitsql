package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func main() {

	log.Println("testing...")
	type Test struct {
		URL  string `csv:"url"`
		Code int    `csv:"code"`
	}

	bb, err := os.ReadFile("./assets/urls.txt")
	if err != nil {
		log.Fatal(errors.Wrap(err, "os.readfile"))
	}
	log.Printf("bytes read: %d", len(bb))
	rdr := csv.NewReader(bytes.NewReader(bb))
	rdr.Comment = '#'
	rdr.FieldsPerRecord = 1
	rdr.TrimLeadingSpace = true
	records, err := rdr.ReadAll()
	if err != nil {
		log.Fatal(errors.Wrap(err, "rdr.readall"))
	}
	log.Printf("records read: %d", len(records))
	tests := []Test{}
	for _, rec := range records {
		if strings.TrimSpace(rec[0]) == "" {
			continue
		}
		test := Test{rec[0], 200}
		tests = append(tests, test)
	}

	for _, t := range tests {
		path := strings.TrimPrefix(t.URL, "/")
		url := fmt.Sprintf("http://localhost:8143/%s", path)
		// println(url)
		code, str, err := getPage(url)
		if err != nil {
			msg := fmt.Sprintf("%s: %s", url, err.Error())
			log.Fatal(msg)
		}
		if code != 200 {
			msg := fmt.Sprintf("ERROR: %s: %s", url, str)
			log.Println(msg)
		}
	}
	log.Printf("tested: %d\n", len(tests))
	log.Println("done.")
}

func getPage(link string) (int, string, error) {
	// println(link)
	resp, err := http.Get(link)
	// println(resp.StatusCode)
	if err != nil {
		return 0, "error", err
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Status, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Status, nil
}
