package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

func main() {
	db, err := sql.Open("sqlserver", "sqlserver://D40/SQL2016")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id      int64
			t1      string
			t2      string
			b       []byte
			ascii   int32
			bytelen int32
			cast    string
			nv      string
		)
		err = rows.Scan(&id, &t1, &t2, &b, &ascii, &bytelen, &nv)
		if err != nil {
			log.Fatal(err)
		}
		cast = string(b)
		fmt.Printf("%d: t1: '%s'  t2: '%s'  len: %d  cast: '%s'  nv: %s\n", id, t1, t2, bytelen, cast, nv)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

var stmt = `
DECLARE @T table (
    id bigint IDENTITY,
    t1 VARCHAR(10),
	t2 NVARCHAR(10),
    b VARBINARY(10) 
)

DECLARE @b as BINARY(1)
DECLARE @s as varchar(1)

Select @b=0x80 --ascii code 128
select @s=convert(varchar(1),@b)

INSERT INTO @T (t1,t2, b) 
VALUES (@s, @s, @b),
		(N'ă', N'ă', CAST(N'ă' AS varbinary(10))),
		(N'日', N'日', CAST(N'日' AS VARBINARY(10))),
		--(N'𒀊', N'𒀊', CAST(N'𒀊' AS VARBINARY(10))),
		--(N'', N'𒀊', CONVERT(VARBINARY, '0x08D80ADC', 1)),
		(N'𒀀', N'𒀀', CONVERT(VARBINARY, '0x08D800DC', 1)),
		(N'𒀀', N'𒀀', CONVERT(VARBINARY, '0x08', 1)),
		(N'𒀀', N'𒀀', CONVERT(VARBINARY, '0x08D8', 1)),
		(N'𒀀', N'𒀀', CONVERT(VARBINARY, '0x08D800', 1)),
		(N'𒀀', N'𒀀', CAST(N'𒀀' AS VARBINARY(10))),
		(N'abcă', N'abcă', CAST(N'abcă' AS VARBINARY(10)))
;
SELECT *
      ,ASCII(t1) as ascii -- 128 -> This value I need to read
	, LEN(b) as [bytes]
	, CONVERT(NVARCHAR(MAX), b) as nv
  FROM @T


 `
