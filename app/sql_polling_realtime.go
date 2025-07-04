package app

import (
	"time"
)

type queryStats struct {
	Database          string
	Object            string
	ObjectShort       string
	StatementShort    string
	Statement         string
	CreationTime      time.Time
	LastExecutionTime time.Time
	ExecutionCount    int64
	AvgWorkerTime     int64
	AvgLogicalReads   int64
}

func (s *SqlServerWrapper) getQueryStats() ([]queryStats, error) {
	var qs []queryStats

	query := `
    
        SET NOCOUNT ON;

        IF OBJECT_ID('tempdb..#T1') IS NOT NULL
            DROP TABLE #T1;

        -- This CTE returns a unified view of the query stats for both in-progress queries (from sys.dm_exec_requests)
        -- and completed queries (from sys.dm_exec_query_stats). 
        ;WITH merged_query_stats AS 
        (
            SELECT 
                [sql_handle], 
                statement_start_offset,
                statement_end_offset,
                plan_generation_num,
                [plan_handle], 
                query_hash AS query_fingerprint, 
                query_plan_hash AS plan_fingerprint, 
                creation_time,
                last_execution_time, 
                execution_count,
                total_worker_time / 1000 AS total_worker_time_ms,
                min_worker_time / 1000 AS min_worker_time_ms,
                max_worker_time / 1000 AS max_worker_time_ms,
                total_physical_reads,
                min_physical_reads,
                max_physical_reads,
                total_logical_writes,
                min_logical_writes,
                max_logical_writes,
                total_logical_reads,
                min_logical_reads,
                max_logical_reads,
                total_clr_time,
                min_clr_time,
                max_clr_time,
                total_elapsed_time / 1000 AS total_elapsed_time_ms,
                min_elapsed_time / 1000 AS min_elapsed_time_ms,
                max_elapsed_time / 1000 AS max_elapsed_time_ms, 
                total_elapsed_time / 1000 AS total_completed_execution_time_ms
            FROM sys.dm_exec_query_stats AS q
            -- To reduce the number of rows that we have to deal with in later queries, filter out any very old rows
            WHERE q.last_execution_time > DATEADD (hour, -4, GETDATE())
            
            -- The UNIONed query below is a workaround for VSTS #91422, sys.dm_exec_query_stats does not reflect stats for in-progress queries. 
            UNION ALL 
            SELECT 
                [sql_handle],
                statement_start_offset,
                statement_end_offset,
                NULL AS plan_generation_num,
                plan_handle,
                query_hash AS query_fingerprint, 
                query_plan_hash AS plan_fingerprint, 
                start_time AS creation_time,
                start_time AS last_execution_time,
                0 AS execution_count,
                cpu_time AS total_worker_time_ms,
                NULL AS min_worker_time_ms,  -- min should not be influenced by in-progress queries
                cpu_time AS max_worker_time_ms,
                reads AS total_physical_reads,
                NULL AS min_physical_reads,  -- min should not be influenced by in-progress queries
                reads AS max_physical_reads,
                writes AS total_logical_writes,
                NULL AS min_logical_writes,  -- min should not be influenced by in-progress queries
                writes AS max_logical_writes,
                logical_reads AS total_logical_reads,
                NULL AS min_logical_reads,   -- min should not be influenced by in-progress queries
                logical_reads AS max_logical_reads,
                NULL AS total_clr_time,      -- CLR time is not available in dm_exec_requests
                NULL AS min_clr_time,        -- CLR time is not available in dm_exec_requests
                NULL AS max_clr_time,        -- CLR time is not available in dm_exec_requests
                total_elapsed_time AS total_elapsed_time_ms,
                NULL AS min_elapsed_time_ms, -- min should not be influenced by in-progress queries
                total_elapsed_time AS max_elapsed_time_ms, 
                NULL AS total_completed_execution_time_ms
            FROM sys.dm_exec_requests AS r 
            WHERE [sql_handle] IS NOT NULL 
                -- Don't attempt to collect stats for very brief in-progress requests; the active statement 
                -- will likely have changed by the time that we harvest query text, in the next query 
                AND DATEDIFF (second, r.start_time, GETDATE()) > 1
        )
        -- Insert the fingerprint stats into a temp table.  SQL isn't always able to produce a good estimate of the amount of 
        -- memory that the upcoming sorts (for ROW_NUMER()) will need because of lack of accurate stats on DMVs.  Staging the 
        -- data in a temp table allows the memory cost of the sort operations to be more accurate, which avoids unnecessary 
        -- spilling to tempdb. 
        , CTE2 AS (

            SELECT 
                fingerprint_stats.*, 
                example_plan.sample_sql_handle, 
                example_plan.sample_plan_handle, 
                example_plan.sample_statement_start_offset, 
                example_plan.sample_statement_end_offset
            --INTO #temp_fingerprint_stats
            FROM
            -- Calculate plan fingerprint stats by grouping the query stats by plan fingerprint
            (
                SELECT 
                    mqs.query_fingerprint, 
                    mqs.plan_fingerprint, 
                    -- The same plan could be returned by both dm_exec_query_stats and dm_exec_requests -- count distinct plan 
                    -- handles only
                    COUNT(DISTINCT plan_handle) AS plan_count, 
                    MIN (mqs.creation_time) AS creation_time, 
                    MAX (mqs.last_execution_time) AS last_execution_time, 
                    SUM (mqs.execution_count) AS execution_count, 
                    SUM (mqs.total_worker_time_ms) AS total_worker_time_ms, 
                    MIN (mqs.min_worker_time_ms) AS min_worker_time_ms, 
                    MAX (mqs.max_worker_time_ms) AS max_worker_time_ms, 
                    SUM (mqs.total_physical_reads) AS total_physical_reads, 
                    MIN (mqs.min_physical_reads) AS min_physical_reads, 
                    MAX (mqs.max_physical_reads) AS max_physical_reads, 
                    SUM (mqs.total_logical_writes) AS total_logical_writes, 
                    MIN (mqs.min_logical_writes) AS min_logical_writes, 
                    MAX (mqs.max_logical_writes) AS max_logical_writes, 
                    SUM (mqs.total_logical_reads) AS total_logical_reads, 
                    MIN (mqs.min_logical_reads) AS min_logical_reads, 
                    MAX (mqs.max_logical_reads) AS max_logical_reads, 
                    SUM (mqs.total_clr_time) AS total_clr_time, 
                    MIN (mqs.min_clr_time) AS min_clr_time, 
                    MAX (mqs.max_clr_time) AS max_clr_time, 
                    SUM (mqs.total_elapsed_time_ms) AS total_elapsed_time_ms, 
                    MIN (mqs.min_elapsed_time_ms) AS min_elapsed_time_ms, 
                    MAX (mqs.max_elapsed_time_ms) AS max_elapsed_time_ms, 
                    SUM (mqs.total_completed_execution_time_ms) AS total_completed_execution_time_ms 
                FROM merged_query_stats AS mqs
                GROUP BY 
                    mqs.query_fingerprint, 
                    mqs.plan_fingerprint
            ) AS fingerprint_stats
            INNER JOIN 
            (
                -- This query assigns a unique row identifier to each plan that has the same fingerprint -- we'll 
                -- select each fingerprint's 'Plan #1' (the earliest example that's still in cache) to use as a sample plan
                -- for the fingerprint.  Later (in the outer query's WHERE clause) we'll filter out all but the first plan, 
                -- and use that one to get a valid sql_handle/plan_handle. 
                SELECT 
                    *, 
                    ROW_NUMBER() OVER (
                        PARTITION BY plan_fingerprint 
                        ORDER BY creation_time 
                    ) AS plan_instance_number 
                FROM 
                (
                    SELECT 
                        query_hash AS query_fingerprint, 
                        query_plan_hash AS plan_fingerprint, 
                        qs.[sql_handle] AS sample_sql_handle, 
                        qs.plan_handle AS sample_plan_handle, 
                        qs.statement_start_offset AS sample_statement_start_offset, 
                        qs.statement_end_offset AS sample_statement_end_offset, 
                        qs.creation_time 
                    FROM sys.dm_exec_query_stats AS qs 
                    -- To get a sample plan for in-progress queries, we need to look in dm_exec_requests, too
                    UNION ALL 
                    SELECT 
                        query_hash AS query_fingerprint, 
                        query_plan_hash AS plan_fingerprint, 
                        r.[sql_handle] AS sample_sql_handle, 
                        r.plan_handle AS sample_plan_handle, 
                        r.statement_start_offset AS sample_statement_start_offset, 
                        r.statement_end_offset AS sample_statement_end_offset, 
                        r.start_time AS creation_time
                    FROM sys.dm_exec_requests AS r
                ) AS all_plans_numbered
            ) AS example_plan 
                ON example_plan.query_fingerprint = fingerprint_stats.query_fingerprint 
                    AND example_plan.plan_fingerprint = fingerprint_stats.plan_fingerprint 
            -- To improve perf of the next query, filter out plan fingerprints that aren't very interesting according to any of our 
            -- perf metrics.  Note that our most frequent allowed execution rate for this script is one execution every 15 seconds, 
            -- so, for example, a plan that is executed 50 times in a 15+ second time interval will qualify for further processing. 
            WHERE plan_instance_number = 1
                AND (fingerprint_stats.total_worker_time_ms > 500       -- 500 ms cumulative CPU time
                OR fingerprint_stats.execution_count > 50               -- 50 executions
                OR fingerprint_stats.total_physical_reads > 50          -- 50 cumulative physical reads
                OR fingerprint_stats.total_logical_reads > 5000         -- 5,000 cumulative logical reads
                OR fingerprint_stats.total_logical_writes > 50          -- 50 cumulative logical writes
                OR fingerprint_stats.total_elapsed_time_ms > 5000)      -- 5 seconds cumulative execution time
            -- SQL doesn't always have good stats on DMVs, and as a result it may select a loop join-based plan w/the 
            -- sys.dm_exec_query_stats DMV as the inner table.  The DMVs don't have indexes that would support efficient 
            -- loop joins, and will commonly have a large enough number of rows that unindexed loop joins will be an  
            -- unattractive option. Given this, we gain much better worst-case perf with minimal cost to best-case perf 
            -- by prohibiting loop joins via this hint. 

        )
        ,CTE2A AS  (
            SELECT *
                ,	total_worker_time_ms / case when execution_count = 0 then 1 else execution_count end as avg_worker_time_ms
                ,	total_logical_reads / case when execution_count = 0 then 1 else execution_count end as avg_logical_reads
                ,	total_logical_writes / case when execution_count = 0 then 1 else execution_count end as avg_logical_writes
            FROM CTE2
        )
        ,CTE3 as (

            -- Add the ranks
            SELECT *,
                ROW_NUMBER () OVER (ORDER BY plan_count DESC) AS plan_count_rank, 
                ROW_NUMBER () OVER (ORDER BY total_worker_time_ms DESC) AS cpu_rank, 
                ROW_NUMBER () OVER (ORDER BY avg_worker_time_ms DESC) AS avg_cpu_rank, 
                ROW_NUMBER () OVER (ORDER BY total_physical_reads DESC) AS physical_reads_rank, 
                ROW_NUMBER () OVER (ORDER BY total_logical_reads DESC) AS logical_reads_rank, 
                ROW_NUMBER () OVER (ORDER BY avg_logical_reads DESC) AS avg_logical_reads_rank, 
                ROW_NUMBER () OVER (ORDER BY total_logical_writes DESC) AS logical_writes_rank, 
                ROW_NUMBER () OVER (ORDER BY max_elapsed_time_ms DESC) AS max_duration_rank, 
                ROW_NUMBER () OVER (ORDER BY execution_count DESC) AS execution_count_rank
            FROM CTE2A 
        )
            SELECT *
            INTO #T1
            FROM CTE3
            OPTION (HASH JOIN, MERGE JOIN);

        --SELECT * FROM #T1 

        SELECT  --top 25 
        --	CAST('<?query -- 
        --	' + batch_text.[text]  + '
        ----?>' AS XML)
            
            COALESCE(plan_info.dbname, '') AS dbname
            ,COALESCE(OBJECT_NAME(objectid, dbid),'') AS ObjectName
            ,COALESCE(SUBSTRING(batch_text.text, (T.sample_statement_start_offset/2)+1,   
                ((CASE T.sample_statement_end_offset  
                WHEN -1 THEN DATALENGTH(batch_text.text)  
                ELSE T.sample_statement_end_offset  
                END - T.sample_statement_start_offset)/2) + 1),'') AS statement_text  
                
            
            --	,batch_text.*
            ,T.creation_time
            ,T.last_execution_time
            ,T.execution_count
            ,T.avg_worker_time_ms
            ,T.avg_logical_reads 
            --,T.avg_cpu_rank
            --,T.avg_logical_reads_rank
            --,T.sample_sql_handle 
                --,T.* 
                --,plan_info.*
        FROM	#T1 AS T
        OUTER APPLY sys.dm_exec_sql_text (sample_sql_handle) AS batch_text
        OUTER APPLY (SELECT DB_NAME(CONVERT(int, value)) AS dbname FROM sys.dm_exec_plan_attributes(sample_plan_handle) WHERE attribute='dbid') AS plan_info
        WHERE	avg_cpu_rank <= 20 or avg_logical_reads_rank <= 20 --or avg_logical_writes <= 20 
        ORDER BY avg_cpu_rank 
                + avg_logical_reads_rank
                --+  avg_logical_writes
                --+ logical_reads_rank 
                --+ max_duration_rank 
                --+ logical_writes_rank  


    
    `

	rows, err := s.DB.Query(query)
	if err != nil {
		WinLogln("Error running query: ", err)
		return qs, err
	}
	defer rows.Close()

	//var qfp, pfp []byte
	for rows.Next() {

		var p queryStats

		err := rows.Scan(&p.Database, &p.Object, &p.Statement, &p.CreationTime, &p.LastExecutionTime,
			&p.ExecutionCount, &p.AvgWorkerTime, &p.AvgLogicalReads)

		if len(p.Statement) > 200 {
			p.StatementShort = p.Statement[:200]
		} else {
			p.StatementShort = ""
		}

		if len(p.Object) > 32 {
			p.ObjectShort = p.Object[:32]
		} else {
			p.ObjectShort = ""
		}

		// fmt.Println(len(p.Object), p.Object)

		if err != nil {
			WinLogln("Error getting rows: ", err)
			return qs, err
		}

		qs = append(qs, p)

	}

	return qs, nil
}
