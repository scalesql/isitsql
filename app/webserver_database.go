package app

// func databasePage(w http.ResponseWriter, req *http.Request) {
// 	mapkey := parms.ByName("server")
// 	wrapper, ok := servers.GetWrapper(mapkey)
// 	if !ok {
// 		renderErrorPage("Invalid Database", fmt.Sprintf("Server Not Found: %s", mapkey), w)
// 		return
// 	}
// 	srv := wrapper.CloneSqlServer()

// 	dbname := parms.ByName("dbid")
// 	db, ok := srv.Databases[dbname]
// 	if !ok {
// 		renderErrorPage("Invalid Database", fmt.Sprintf("Not Found: Server: %s  Database: %s", mapkey, dbname), w)
// 		return
// 	}

// 	query := fmt.Sprintf("EXEC sp_executesql N'USE [%s];\r\n\r\n", db.Name)
// 	query += `
// 			SELECT
// 			--(row_number() over(order by (a1.reserved + ISNULL(a4.reserved,0)) desc))%2 as l1,
// 			a3.name AS [schemaname],
// 			a2.name AS [tablename],
// 			a1.rows as row_count,
// 			(a1.reserved + ISNULL(a4.reserved,0))* 8 AS reserved,
// 			a1.data * 8 AS data,
// 			(CASE WHEN (a1.used + ISNULL(a4.used,0)) > a1.data THEN (a1.used + ISNULL(a4.used,0)) - a1.data ELSE 0 END) * 8 AS index_size,
// 			(CASE WHEN (a1.reserved + ISNULL(a4.reserved,0)) > a1.used THEN (a1.reserved + ISNULL(a4.reserved,0)) - a1.used ELSE 0 END) * 8 AS unused
// 			FROM
// 			(SELECT
// 			ps.object_id,
// 			SUM (
// 			CASE
// 			WHEN (ps.index_id < 2) THEN row_count
// 			ELSE 0
// 			END
// 			) AS [rows],
// 			SUM (ps.reserved_page_count) AS reserved,
// 			SUM (
// 			CASE
// 			WHEN (ps.index_id < 2) THEN (ps.in_row_data_page_count + ps.lob_used_page_count + ps.row_overflow_used_page_count)
// 			ELSE (ps.lob_used_page_count + ps.row_overflow_used_page_count)
// 			END
// 			) AS data,
// 			SUM (ps.used_page_count) AS used
// 			FROM sys.dm_db_partition_stats ps
// 			WHERE ps.object_id NOT IN (SELECT object_id FROM sys.tables WHERE is_memory_optimized = 1)
// 			GROUP BY ps.object_id) AS a1
// 			LEFT OUTER JOIN
// 			(SELECT
// 			it.parent_id,
// 			SUM(ps.reserved_page_count) AS reserved,
// 			SUM(ps.used_page_count) AS used
// 			FROM sys.dm_db_partition_stats ps
// 			INNER JOIN sys.internal_tables it ON (it.object_id = ps.object_id)
// 			WHERE it.internal_type IN (202,204)
// 			GROUP BY it.parent_id) AS a4 ON (a4.parent_id = a1.object_id)
// 			INNER JOIN sys.all_objects a2  ON ( a1.object_id = a2.object_id )
// 			INNER JOIN sys.schemas a3 ON (a2.schema_id = a3.schema_id)
// 			WHERE a2.type <> N''S'' and a2.type <> N''IT''
// 			'
// 	`

// 	type table struct {
// 		Schema   string
// 		Name     string
// 		Rows     int64
// 		Reserved int64
// 		Data     int64
// 		Index    int64
// 		Unused   int64
// 	}
// 	tables := make([]table, 0)

// 	rows, err := wrapper.DB.Query(query)
// 	if err != nil {
// 		renderErrorPage("Error Querying Tables", "Error querying tables", w)
// 		WinLogln(err.Error())
// 		return
// 	}

// 	for rows.Next() {
// 		var t table
// 		err := rows.Scan(&t.Schema, &t.Name, &t.Rows, &t.Reserved, &t.Data, &t.Index, &t.Unused)
// 		if err != nil {
// 			renderErrorPage("Error Querying Tables", "Scan Error", w)
// 			WinLogln(err.Error())
// 			return
// 		}
// 		tables = append(tables, t)
// 	}
// 	err = rows.Err()
// 	if err != nil {
// 		renderErrorPage("Error Querying Tables", "Error querying tables", w)
// 		WinLogln(err.Error())
// 		return
// 	}

// 	var pd struct {
// 		Context
// 		Database Database
// 		Tables   []table
// 	}

// 	pd.Context = getContext(fmt.Sprintf("%s - %s", db.Name, srv.ServerName))
// 	pd.Context.OneServer = &srv
// 	pd.Tables = tables
// 	pd.Database = *db

// 	renderFSDynamic(w, "database", pd)
// }
