-- latest execution completed for each job
--SELECT  * from msdb.dbo.sysjobhistory where step_id = 0 
-- last completion time for each job
;WITH last_exec AS (
	SELECT   
		ROW_NUMBER() OVER( partition by job_id ORDER BY instance_id DESC) as rownum
		, *
	FROM msdb.dbo.sysjobhistory
	WHERE step_id = 0 
)
SELECT * FROM last_exec
WHERE rownum = 1
order by job_id 

-- Currently running steps for a job 
SELECT * from msdb.dbo.sysjobhistory
where job_id = 'D07B632F-ECF6-46F3-907C-475579AA53DA'
and instance_id > (SELECT MAX(instance_id) FROM msdb.dbo.sysjobhistory WHERE job_id = 'D07B632F-ECF6-46F3-907C-475579AA53DA' AND step_id = 0)
order by step_id, retries_attempted

-- next scheduled run_date for all jobs 
SELECT * 
	,CASE 
		WHEN run_requested_date IS NOT NULL AND start_execution_date IS NOT NULL AND stop_execution_date IS NULL THEN 1 
		ELSE 0
	END AS is_running
FROM msdb.dbo.sysjobactivity 
WHERE session_id IN (SELECT MAX(session_id) FROM msdb.dbo.sysjobactivity )
order by job_id

SELECT  jv.* 
		,COALESCE(SUSER_SNAME(owner_sid), '') as owner_name 
		,COALESCE(c.[name], '') AS category_name
FROM msdb.dbo.sysjobs_view jv
LEFT JOIN msdb.dbo.syscategories c ON c.category_id = jv.category_id
ORDER BY job_id 

--select * from syscategories order by [name]
-------------- FINAL ------------------------
;WITH completions AS (
	SELECT   
		ROW_NUMBER() OVER( partition by job_id ORDER BY instance_id DESC) as rownum
		, *
	FROM msdb.dbo.sysjobhistory
	WHERE step_id = 0 
), last_exec AS (
	SELECT * FROM completions
	WHERE rownum = 1
), activity AS (
	SELECT * 
		,CASE 
			WHEN run_requested_date IS NOT NULL AND start_execution_date IS NOT NULL AND stop_execution_date IS NULL THEN 1 
			ELSE 0
		END AS is_running
	FROM msdb.dbo.sysjobactivity 
	WHERE session_id IN (SELECT MAX(session_id) FROM msdb.dbo.sysjobactivity )
)
-- TODO: get the current retry count from sysjobhistory after our last completed execution
--, history AS (
--	SELECT * from msdb.dbo.sysjobhistory
--	--where job_id = 'D07B632F-ECF6-46F3-907C-475579AA53DA'
--	and instance_id > (SELECT MAX(instance_id) FROM msdb.dbo.sysjobhistory WHERE job_id = 'D07B632F-ECF6-46F3-907C-475579AA53DA' AND step_id = 0)
--	order by step_id, retries_attempted
--)
SELECT  jv.* 
		,COALESCE(SUSER_SNAME(owner_sid), '') as owner_name 
		,COALESCE(c.[name], '') AS category_name
		,COALESCE(last_exec.run_status, -1) as run_status
		,COALESCE(last_exec.run_date, 0) AS run_date
		,COALESCE(last_exec.run_time , 0) AS run_time
		,COALESCE(last_exec.run_duration , 0) AS run_duration
		,activity.is_running
		,activity.start_execution_date
		,activity.next_scheduled_run_date
FROM msdb.dbo.sysjobs_view jv
LEFT JOIN msdb.dbo.syscategories c ON c.category_id = jv.category_id
LEFT JOIN last_exec ON last_exec.job_id = jv.job_id
LEFT JOIN activity ON activity.job_id = jv.job_id
ORDER BY job_id 
