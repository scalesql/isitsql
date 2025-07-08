-- +goose Up
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;

CREATE TABLE [dbo].[server_metric](
	[metric_time] [datetimeoffset](0) NOT NULL,
	[server_key] [nvarchar](128) NOT NULL,
    [server_name] [nvarchar](128) NOT NULL,
	[cpu_cores] smallint NULL, 
	[cpu_sql_pct] [tinyint] NULL,
	[cpu_other_pct] [tinyint] NULL,
	[batches_per_second] int NULL,
	page_life_expectancy int NULL,
	memory_used_mb INT NULL,
	disk_read_iops INT NULL,
	disk_read_kb_sec INT NULL,
	disk_read_latency_ms INT NULL,
	disk_write_iops INT NULL,
	disk_write_kb_sec INT NULL,
	disk_write_latency_ms INT NULL
) ON [PRIMARY];


CREATE CLUSTERED COLUMNSTORE INDEX [ccx_metric] ON [dbo].[server_metric] 
	WITH (DROP_EXISTING = OFF, COMPRESSION_DELAY = 0) ON [PRIMARY];

-- +goose Down
DROP TABLE [dbo].[server_metric];