package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/billgraziano/mssqlh/v2"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
	"github.com/pkg/errors"
)

func main() {
	var fqdn string
	if len(os.Args) < 2 {
		log.Println("usage: conntest.exe fqdn")
		return
	}
	fqdn = os.Args[1]
	if fqdn == "" {
		log.Println("usage: conntest.exe fqdn")
		return
	}
	log.Printf("fqdn: %s\n", fqdn)

	conn := mssqlh.NewConnection(fqdn, "", "", "", "conntest.exe")
	log.Printf("connection string: %s", conn.String())

	db, err := sql.Open("sqlserver", conn.String())
	if err != nil {
		log.Fatal(errors.Wrap(err, "sql.open"))
	}
	defer db.Close()

	var serverName string
	err = db.QueryRow("SELECT @@SERVERNAME").Scan(&serverName)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("@@SERVERNAME: %s\n", serverName)

	var auth, iface, encrypt string
	var version int64

	stmt := `
		SELECT	c.auth_scheme, s.client_version, s.client_interface_name, c.encrypt_option
		FROM	sys.dm_exec_sessions s
		JOIN	sys.dm_exec_connections c ON c.session_id = s.session_id
		WHERE 	s.session_id = @@SPID;
	`
	err = db.QueryRow(stmt).Scan(&auth, &version, &iface, &encrypt)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("auth: '%s';  encrypt: '%s';  interface: '%s';  version: %d", auth, encrypt, iface, version)
}
