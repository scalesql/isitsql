-- +goose Up
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;

CREATE TABLE [dbo].[server_wait](
	[metric_time] [datetimeoffset](0) NOT NULL,
	[server_key] [nvarchar](128) NOT NULL,
    [server_name] [nvarchar](128) NOT NULL,
    [wait_type] NVARCHAR(128) NOT NULL,
    [wait_time_ms] BIGINT NOT NULL,
) ON [PRIMARY];


CREATE CLUSTERED COLUMNSTORE INDEX [ccx_server_wait] ON [dbo].[server_wait] 
	WITH (DROP_EXISTING = OFF, COMPRESSION_DELAY = 0) ON [PRIMARY];

-- +goose Down
DROP TABLE [dbo].[server_wait];