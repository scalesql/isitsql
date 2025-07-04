package app

import (
	"encoding/json"
	"net/http"
)

func ApiSizes(w http.ResponseWriter, r *http.Request) {

	type GraphData struct {
		Object string
		Parent string
		SizeKB int64
	}

	type GraphResult struct {
		Headers []string
		Values  []GraphData
	}

	//log.Println("Call to ApiSizes...")

	var gr GraphResult
	gr.Headers = []string{"Object", "Parent", "Size"}

	gd1 := GraphData{"Parent", "null", 0}
	gd2 := GraphData{"DB1", "Parent", 37}

	gr.Values = []GraphData{gd1, gd2}

	json.NewEncoder(w).Encode(gr)

}
