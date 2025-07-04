# BUCKET

```go
type WaitBuffer struct {
    MapKey string
    Waits *Waits 
}
```

```go
type BucketWriter struct {
    mu sync.RWMutex
    string prefix
    file system
    clock
    file descriptor
    buffered channel (200?)
    closed bool // check if we are closed???
    rollDuration time.Duration // 10 minutes
    retainDuration time.Duration // 100 minutes
}

Start(path) (BucketWriter, error) // build this and start the goroutine
Enqueue(Server GUID, interface) (error) // exit if closed, send on channel, timeout after 1 second
Stop()  // close the channel to allow to drain
GoRoutine() {
    // clean up old files
    // start a file (aka roll file)
    // read channel and write file
        // roll file every x minutes
    // if cancel (or close), drain and exit
}
```

```go
type BucketReader struct {
    string prefix
    fs afero.Fs
    files []string
    int currentFile //pointer to current file
    // Reader for the current file
}

NewReader(path) (BucketReader, error)
Next() (string, bytes, error) // read the current file, roll to the next one
// NextWaits (Waits, error) // wrapper for Next()
```

## Notes
* NDJSON rolling every 10 mintues
* struct with GUID and then the metric value
* Roll the file every 10 minutes with UTC time stamp
* Header - magic text, file version (int), header length (int), header payload (JSON), etc.
* Each record is as follows:
  * length (int)
  * payload (bytes) -- this is JSON from each metrics, CPU, disk, etc.
  * A length of zero is the end of the file.  Or just the end of the file. 

## Reader
* New creates the bucket structure
* 