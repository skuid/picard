CREATE TABLE picklist_entry
(
  
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT picklist_entry_pkey
    PRIMARY KEY,
  organization_id       varchar(36)                            NOT NULL,
    --CONSTRAINT picklist_entry_organization_id_foreign
    --REFERENCES organization,
  created_by_id         varchar(36)                            NOT NULL,
    --CONSTRAINT picklist_entry_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         varchar(36)                            NOT NULL,
    --CONSTRAINT picklist_entry_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  data_source_field_id UUID                            NOT NULL
    CONSTRAINT picklist_entry_data_source_field_id_foreign
    REFERENCES data_source_field
    ON DELETE CASCADE,
  active                BOOLEAN,
  value                 VARCHAR(255),
  label                 VARCHAR(255)
);
