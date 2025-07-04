package main

import (
	"database/sql"
	"log"

	_ "github.com/alexbrainman/odbc"
	"github.com/billgraziano/mssqlodbc"
)

func main() {
	cxn := mssqlodbc.Connection{
		Server:              "D40\\SQL2016",
		Database:            "tempdb",
		AppName:             "gosql",
		Trusted:             true,
		MultiSubnetFailover: true,
	}

	s, err := cxn.ConnectionString()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("odbc", s)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
