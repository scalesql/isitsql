package main

import (
	"database/sql"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/billgraziano/mssqlh/v2"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
	"github.com/pkg/errors"
)

var wg sync.WaitGroup

func main() {

	wg.Add(1)
	go func() {
		log.Println("Starting blocker...")
		conn := mssqlh.NewConnection("D40\\SQL2016", "", "", "", "makeblock.exe")
		log.Printf("connection string: %s", conn.String())

		db, err := sql.Open("sqlserver", conn.String())
		if err != nil {
			log.Fatal(errors.Wrap(err, "sql.open"))
		}
		defer db.Close()

		//var serverName string
		_, err = db.Exec(`
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
		BEGIN TRAN 
			SELECT TOP 10 *     FROM	AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
			WAITFOR DELAY '00:59:00'
		ROLLBACK TRAN `)
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("@@SERVERNAME: %s\n", serverName)
		err = db.Close()
		if err != nil {
			log.Print(err)
		}
		wg.Done()
	}()
	time.Sleep(1 * time.Second)
	for i := 1; i <= 500; i++ {
		time.Sleep(10 * time.Millisecond)
		wg.Add(1)
		go func(x int) {
			runtime.LockOSThread()
			log.Printf("go: %d\n", x)
			conn := mssqlh.NewConnection("D40\\SQL2016", "", "", "", "makeblock.exe")
			db, err := sql.Open("sqlserver", conn.String())
			if err != nil {
				log.Fatal(errors.Wrap(err, "sql.open"))
			}
			defer db.Close()

			//var serverName string
			_, err = db.Exec(`
			WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
			BEGIN TRAN 
				SELECT TOP 10 *     FROM	AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
			ROLLBACK TRAN `)
			if err != nil {
				log.Fatal(err)
			}
			// log.Printf("@@SERVERNAME: %s\n", serverName)
			err = db.Close()
			if err != nil {
				log.Print(err)
			}
			wg.Done()
			runtime.UnlockOSThread()
		}(i)
	}
	wg.Wait()
}
