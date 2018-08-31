ALTER TABLE site_jwt_key
	ALTER COLUMN organization_id TYPE varchar(36),
	ALTER COLUMN updated_by_id TYPE varchar(36),
	ALTER COLUMN created_by_id TYPE varchar(36),
	ADD CONSTRAINT organization_id_unique
		unique (organization_id);