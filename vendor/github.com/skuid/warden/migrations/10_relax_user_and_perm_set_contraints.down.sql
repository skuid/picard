
ALTER TABLE data_source
	ALTER COLUMN auth_provider_id TYPE UUID,
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_permission
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN permission_set_id TYPE UUID;

ALTER TABLE data_source_credential
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN user_id TYPE UUID;

ALTER TABLE data_source_object
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_object_permission
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN permission_set_id TYPE UUID;

ALTER TABLE data_source_condition
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_condition_permission
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN permission_set_id TYPE UUID;

ALTER TABLE data_source_field
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_field_permission
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN permission_set_id TYPE UUID;

ALTER TABLE data_source_action
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_action_input
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_action_output
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID;

ALTER TABLE data_source_action_permission
	ALTER COLUMN updated_by_id TYPE UUID,
	ALTER COLUMN created_by_id TYPE UUID,
	ALTER COLUMN permission_set_id TYPE UUID;
