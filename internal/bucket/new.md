New Wait Store
==============
* Just for writing server waits
* Methods: Write(waits), Read() []Waits
* Locking done in the file write
* Zero value of the struct. Do I need a struct?
* Used by PollWaits()
* `waits.store.key.time_stamp.ndjson`
    * maybe just reuse the existing file names?
* bucket writer is used for writing new waits to a central store
* this writes old waits to a per server store

