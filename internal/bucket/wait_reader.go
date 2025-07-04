package bucket

// ReadWaits reads the cached waits in the files.
// It reads the server specific wait files.  It reads all the files for a server
// and filters for the key it wants.  Yes, this is redudant.
// It runs in about 50ms on a DEV box.
// This should be fairly fast as these files will be cached in memory.
// The only page that uses it is /server/:key/waits
// This allows us to remove the WaitRing from the SQL Server structure.
// We hope this drastically reduces memory.
// func (bw *BucketWriter) ReadWaits(key string) ([]waitmap.Waits, error) {
// 	results := make([]waitmap.Waits, 0, 240)
// 	defer failure.HandlePanic()

// 	// get the files
// 	start := time.Now()
// 	// fmt.Sprintf("%s/bw.%s.%s.%s.ndjson", bw.path, bw.prefix, mapkey, bw.clock.Now().UTC().Round(10*time.Minute).Format("20060102_1504"))
// 	pattern := filepath.Join(bw.path, fmt.Sprintf("bw.%s.%s.*.ndjson", bw.prefix, key))
// 	files, err := afero.Glob(bw.fs, pattern)
// 	if err != nil {
// 		return results, errors.Wrap(err, "glob")
// 	}
// 	sort.Strings(files)
// 	globDuration := time.Since(start)

// 	if globDuration > time.Duration(100*time.Millisecond) {
// 		logrus.Debugf("readwaits: glob: files: %d  (%s)", len(files), globDuration)
// 	}

// 	// process each file
// 	start = time.Now()
// 	var totalLines int
// 	for _, fileName := range files {
// 		// read the entire file into memory
// 		bb, err := afero.ReadFile(bw.fs, fileName)
// 		if err != nil {
// 			return results, errors.Wrap(err, "afero.readfile")
// 		}

// 		// read each line of the NDJSON
// 		// the goal is to return an array of waitmap.Waits
// 		for _, v := range bytes.Split(bb, []byte{'\n'}) {
// 			line := v
// 			// it seems to be returning a zero length line at the end
// 			if len(line) == 0 {
// 				continue
// 			}

// 			totalLines++
// 			wl := new(WaitLine)
// 			err = json.Unmarshal(line, &wl)
// 			if err != nil {
// 				return results, errors.Wrap(err, "json.unmarshal")
// 			}
// 			if wl.MapKey == key {
// 				results = append(results, wl.Payload)
// 			}
// 		}
// 	}
// 	if time.Since(start) > time.Duration(100*time.Millisecond) {
// 		logrus.Debugf("readwaits: read: read=%d  kept=%d  (%s)", totalLines, len(results), time.Since(start))
// 	}
// 	return results, nil
// }

// type WaitLine struct {
// 	MapKey  string        `json:"map_key"`
// 	Payload waitmap.Waits `json:"payload"`
// }
