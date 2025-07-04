package main

import (
	"database/sql"
	"log"
	"net/url"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

func main() {
	//duation := "00:00:10"
	log.Println("starting...")
	query := url.Values{}
	u := &url.URL{
		Scheme:   "sqlserver",
		Host:     "localhost",
		Path:     "SQL2016", // if connecting to an instance instead of a port
		RawQuery: query.Encode(),
	}
	// db, err := sql.Open("sqlserver", u.String())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()
	statements := make([]string, 0)
	statements = append(statements, `
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
		BEGIN TRAN 
			SELECT TOP 10 *     FROM	AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
			-- SELECT TOP 10 * FROM    AdventureWorks2016.Sales.SalesOrderHeader WITH(UPDLOCK, TABLOCK);
			WAITFOR DELAY '00:15:00'
		ROLLBACK TRAN `)

	statements = append(statements, `
		-- Session 2 
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
			BEGIN TRAN 
			SELECT TOP 10 * FROM AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
		ROLLBACK TRAN `)

	statements = append(statements, `
		-- Session 3
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
		BEGIN TRAN 
			SELECT TOP 10 * FROM    AdventureWorks2016.Sales.SalesOrderHeader WITH(UPDLOCK, TABLOCK);
			SELECT TOP 10 * FROM	AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
		ROLLBACK TRAN `)

	statements = append(statements, `
		-- Session 4
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
		--BEGIN TRAN 
		SELECT TOP 10 * FROM    AdventureWorks2016.Sales.SalesOrderHeader WITH(UPDLOCK, TABLOCK);`)

	statements = append(statements, `
		-- Session 6
		WHILE @@TRANCOUNT > 0 ROLLBACK TRAN ;
		BEGIN TRAN 
		SELECT TOP 10 * FROM	AdventureWorks2016.Person.Person WITH(UPDLOCK, TABLOCK);
		ROLLBACK TRAN`)

	var wg sync.WaitGroup
	log.Printf("statements=%d\n", len(statements))
	for i, s := range statements {
		wg.Add(1)
		log.Printf("starting: #%d\n", i+1)
		go func(cxn string, stmt string) {
			defer wg.Done()
			// println(s)
			db, err := sql.Open("sqlserver", u.String())
			if err != nil {
				log.Println(err.Error())
				return
			}
			//defer db.Close()
			_, err = db.Exec(s)
			if err != nil {
				log.Println(err.Error())
			}
		}(u.String(), s)
		time.Sleep(1 * time.Second)
	}
	log.Println("waiting...")
	wg.Wait()
}
