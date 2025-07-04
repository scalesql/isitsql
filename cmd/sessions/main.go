package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"

	"github.com/scalesql/isitsql/internal/mssql/session"
)

func main() {
	query := url.Values{}
	//query.Add("app name", "MyAppName")

	u := &url.URL{
		Scheme: "sqlserver",
		//User:   url.UserPassword(username, password),
		Host:     "localhost",
		Path:     "SQL2016", // if connecting to an instance instead of a port
		RawQuery: query.Encode(),
	}
	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		log.Fatal(err)
	}
	sessions, err := session.Get(context.Background(), db, 11)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", sessions)
	for _, s := range sessions {
		fmt.Printf("id=%d  request=%v  blocker=%-2d  head=%-2d  blocking=%-2d\n", s.SessionID, s.HasRequest, s.BlockerID, s.HeadBlockerID, s.TotalBlocked)
	}
}
