-- This constraint is not needed. It does not exist in the corollary table in Pliny.
ALTER TABLE data_source DROP CONSTRAINT data_source_authentication_check;
ALTER TABLE data_source DROP CONSTRAINT data_source_credential_source_check;
ALTER TABLE data_source
	-- Add new columns for allowing for specifying how to retrieve temporary access credentials.
	ADD COLUMN get_credentials_access_key_id TEXT,
	ADD COLUMN get_credentials_secret_access_key TEXT,
	ADD COLUMN credential_duration_seconds INTEGER,
	ADD COLUMN credential_username_source_type VARCHAR(50),
	ADD COLUMN credential_username_source_property VARCHAR(255),
	ADD COLUMN credential_groups_source_type VARCHAR(50),
	ADD COLUMN credential_groups_source_property VARCHAR(255);