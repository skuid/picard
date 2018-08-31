ALTER TABLE data_source
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_permission
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_credential
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_object
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_object_permission
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_condition
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_condition_permission
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_field
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_field_permission
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_action
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_action_input
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_action_output
	ALTER COLUMN organization_id TYPE UUID;

ALTER TABLE data_source_action_permission
	ALTER COLUMN organization_id TYPE UUID;