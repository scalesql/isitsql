-- +goose Up
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;

IF NOT EXISTS(  SELECT * 
                FROM INFORMATION_SCHEMA.COLUMNS 
                WHERE TABLE_SCHEMA = 'dbo' 
                AND TABLE_NAME = 'server_metric' 
                AND COLUMN_NAME='server_start')
    ALTER TABLE dbo.[server_metric] ADD [server_start] DATETIME NULL;

IF NOT EXISTS(  SELECT * 
                FROM INFORMATION_SCHEMA.COLUMNS 
                WHERE TABLE_SCHEMA = 'dbo' 
                AND TABLE_NAME = 'request_wait' 
                AND COLUMN_NAME='server_start')
    ALTER TABLE dbo.[request_wait] ADD [server_start] DATETIME NULL;

IF NOT EXISTS(  SELECT * 
                FROM INFORMATION_SCHEMA.COLUMNS 
                WHERE TABLE_SCHEMA = 'dbo' 
                AND TABLE_NAME = 'server_wait' 
                AND COLUMN_NAME='server_start')
    ALTER TABLE dbo.[server_wait] ADD [server_start] DATETIME NULL;


-- +goose Down
ALTER TABLE dbo.server_metric DROP COLUMN [server_start];
ALTER TABLE dbo.request_wait DROP COLUMN [server_start];
ALTER TABLE dbo.server_wait DROP COLUMN [server_start];