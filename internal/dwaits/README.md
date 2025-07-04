Dynamic Real-Time Waits
====================
* w2.Repository is defined as an `app` level variable.  A pointer to the Repository is passed into each server for the box.  The box calls `&Repository.Enqueue(key, waits)`.  And commented out for JSON.
* Create memory page that shows the sizes of various objects
    * w2 Repository
    * Servers total and average 
        * Objects inside?  Databases?
    * Backups/AGs?

## Repository
* mu - protects adds and deletes
* Servers - map[map_key]
    * Another lock?
    * wait ring
* `bucket`
* open bool 
* Context

### Methods
* Repositry.New(...) (*Repository, error)
    * Lock()
    * Read the files and populate the history
    * Ignore more than 1 hour old
* Close() 
    * Lock, active=false, flush, close bucket
* Write()
    * Lock
    * Calls enqueue() - writes to `bucket`
        * If not active, return
* Waits(key) - returns the waits for a server
    * RLock()
* Add(key, ...)
* Remove(key, ...)

## Features
* The Repository can purge any server that is more than one hour old?  Maintenance?  Will need a stopable GO routine





