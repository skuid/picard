-- This was run before we were storing real credentials in data_source table.
-- Deleting all records from data_source creates a blank slate, so that all
--  future data_source records have encrypted credential columns.
TRUNCATE public.data_source CASCADE;