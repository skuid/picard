-- add back the data_source_authentication_check constraint
ALTER TABLE data_source ADD CONSTRAINT data_source_authentication_check
	CHECK (authentication = ANY (ARRAY ['none' :: TEXT, 'basicauth' :: TEXT, 'separateurl' :: TEXT, 'provider' :: TEXT]));
-- add back the data_source_credential_source_check constraint
ALTER TABLE data_source ADD CONSTRAINT data_source_credential_source_check
    CHECK (credential_source = ANY (ARRAY ['none' :: TEXT, 'org' :: TEXT, 'user' :: TEXT]));
-- remove new columns added by the up
ALTER TABLE data_source
	DROP COLUMN get_credentials_access_key_id,
	DROP COLUMN get_credentials_secret_access_key,
	DROP COLUMN credential_duration_seconds,
	DROP COLUMN credential_username_source_type,
	DROP COLUMN credential_username_source_property,
	DROP COLUMN credential_groups_source_type,
	DROP COLUMN credential_groups_source_property;