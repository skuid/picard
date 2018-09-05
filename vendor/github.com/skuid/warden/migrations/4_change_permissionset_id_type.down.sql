ALTER TABLE data_source_permission
ALTER COLUMN permission_set_id TYPE UUID;
ALTER TABLE data_source_object_permission
ALTER COLUMN permission_set_id TYPE UUID;
ALTER TABLE data_source_condition_permission
ALTER COLUMN permission_set_id TYPE UUID;
ALTER TABLE data_source_field_permission
ALTER COLUMN permission_set_id TYPE UUID;
ALTER TABLE data_source_action_permission
ALTER COLUMN permission_set_id TYPE UUID;