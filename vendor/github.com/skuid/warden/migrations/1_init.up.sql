-- Add the uuid extension if it doesn't exist already
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- auto-generated definition
CREATE TABLE data_source
(
  name                           VARCHAR(255)                    NOT NULL,
  authentication                 TEXT DEFAULT 'none' :: TEXT     NOT NULL
    CONSTRAINT data_source_authentication_check
    CHECK (authentication = ANY
           (ARRAY ['none' :: TEXT, 'basicauth' :: TEXT, 'separateurl' :: TEXT, 'provider' :: TEXT])),
  type                           TEXT DEFAULT 'REST' :: TEXT     NOT NULL,
  credential_source              TEXT        DEFAULT 'none' :: TEXT
    CONSTRAINT data_source_credential_source_check
    CHECK (credential_source = ANY (ARRAY ['none' :: TEXT, 'org' :: TEXT, 'user' :: TEXT])),
  url                            VARCHAR(1000),
  is_active                      BOOLEAN     DEFAULT TRUE,
  --use_proxy                      BOOLEAN     DEFAULT TRUE,
  auth_url                       VARCHAR(1000),
  auth_request_body              JSONB,
  auth_request_body_content_type TEXT
    CONSTRAINT data_source_auth_request_body_content_type_check
    CHECK (auth_request_body_content_type = ANY
           (ARRAY ['application/json' :: TEXT, 'application/x-www-form-urlencoded' :: TEXT])),
  auth_request_headers           JSONB,
  auth_request_verb              TEXT
    CONSTRAINT data_source_auth_request_verb_check
    CHECK (auth_request_verb = ANY (ARRAY ['GET' :: TEXT, 'POST' :: TEXT])),
  request_body_parameters        JSONB,
  request_headers                JSONB,
  request_url_parameters         JSONB,
  request_scopes                 VARCHAR(255),
  username                       VARCHAR(500),
  password                       VARCHAR(500),
  created_at                     TIMESTAMP WITH TIME ZONE,
  updated_at                     TIMESTAMP WITH TIME ZONE,
  config                         JSONB,
  name_identifier_field          VARCHAR(20) DEFAULT 'federation_id' :: CHARACTER VARYING,
  name_identifier_formula        VARCHAR(1000),
  metadata_cache_version         BIGINT      DEFAULT '0' :: BIGINT,
  metadata_cache_key             VARCHAR(255),
  database_type                  VARCHAR(25),
  database_name                  VARCHAR(1000),
  database_username              VARCHAR(1000),
  database_password              VARCHAR(1000),
  id                             UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_pkey
    PRIMARY KEY,
  auth_provider_id               UUID,
    --CONSTRAINT data_source_auth_provider_id_foreign
    --REFERENCES auth_provider,
  updated_by_id                  UUID,
    --CONSTRAINT data_source_updated_by_id_foreign
    --REFERENCES "user",
  created_by_id                  UUID,
    --CONSTRAINT data_source_created_by_id_foreign
    --REFERENCES "user",
  organization_id                UUID,
    --CONSTRAINT data_source_organization_id_foreign
    --REFERENCES organization,
  --use_data_source_objects        BOOLEAN     DEFAULT FALSE,
  CONSTRAINT data_source_name_organization_id_unique
  UNIQUE (name, organization_id)
);

-- auto-generated definition
CREATE TABLE data_source_permission
(
  created_at        TIMESTAMP WITH TIME ZONE,
  updated_at        TIMESTAMP WITH TIME ZONE,
  data_source_id    UUID
    CONSTRAINT data_source_permission_data_source_id_foreign
    REFERENCES data_source
    ON DELETE CASCADE,
  id                UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_permission_pkey
    PRIMARY KEY,
  updated_by_id     UUID,
    --CONSTRAINT data_source_permission_updated_by_id_foreign
    --REFERENCES "user",
  created_by_id     UUID,
    --CONSTRAINT data_source_permission_created_by_id_foreign
    --REFERENCES "user",
  permission_set_id UUID,
    --CONSTRAINT data_source_permission_permission_set_id_foreign
    --REFERENCES permission_set
    --ON DELETE CASCADE,
  organization_id   UUID,
    --CONSTRAINT data_source_permission_organization_id_foreign
    --REFERENCES organization,
  CONSTRAINT data_source_permission_data_source_id_permission_set_id_unique
  UNIQUE (data_source_id, permission_set_id)
);

-- auto-generated definition
CREATE TABLE data_source_credential
(
  username                VARCHAR(255),
  password                VARCHAR(255),
  access_token            VARCHAR(255),
  refresh_token           VARCHAR(255),
  access_token_expiration TIMESTAMP WITH TIME ZONE,
  created_at              TIMESTAMP WITH TIME ZONE,
  updated_at              TIMESTAMP WITH TIME ZONE,
  data_source_id          UUID
    CONSTRAINT data_source_credential_data_source_id_foreign
    REFERENCES data_source,
  id                      UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_credential_pkey
    PRIMARY KEY,
  user_id                 UUID,
    --CONSTRAINT data_source_credential_user_id_foreign
    --REFERENCES "user",
  updated_by_id           UUID,
    --CONSTRAINT data_source_credential_updated_by_id_foreign
    --REFERENCES "user",
  created_by_id           UUID,
    --CONSTRAINT data_source_credential_created_by_id_foreign
    --REFERENCES "user",
  organization_id         UUID
    --CONSTRAINT data_source_credential_organization_id_foreign
    --REFERENCES organization
);


-- auto-generated definition
CREATE TABLE data_source_object
(
  id              UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_object_pkey
    PRIMARY KEY,
  organization_id UUID                            NOT NULL,
    --CONSTRAINT data_source_object_organization_id_foreign
    --REFERENCES organization,
  created_by_id   UUID                            NOT NULL,
    --CONSTRAINT data_source_object_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id   UUID                            NOT NULL,
    --CONSTRAINT data_source_object_updated_by_id_foreign
    --REFERENCES "user",
  created_at      TIMESTAMP WITH TIME ZONE,
  updated_at      TIMESTAMP WITH TIME ZONE,
  data_source_id  UUID                            NOT NULL
    CONSTRAINT data_source_object_data_source_id_foreign
    REFERENCES data_source,
  name            VARCHAR(255)                    NOT NULL,
  label           VARCHAR(255),
  label_plural    VARCHAR(255),
  custom_config   JSONB
);

-- auto-generated definition
CREATE TABLE data_source_object_permission
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_object_permission_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_object_permission_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_object_permission_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_object_permission_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  permission_set_id     UUID                            NOT NULL,
    --CONSTRAINT data_source_object_permission_permission_set_id_foreign
    --REFERENCES permission_set
    --ON DELETE CASCADE,
  data_source_object_id UUID                            NOT NULL
    CONSTRAINT data_source_object_permission_data_source_object_id_foreign
    REFERENCES data_source_object
    ON DELETE CASCADE,
  createable            BOOLEAN,
  deleteable            BOOLEAN,
  queryable             BOOLEAN,
  updateable            BOOLEAN,
  CONSTRAINT data_source_object_permission_permission_set_id_data_source_obj
  UNIQUE (permission_set_id, data_source_object_id)
);


-- auto-generated definition
CREATE TABLE data_source_condition
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_condition_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  data_source_object_id UUID                            NOT NULL
    CONSTRAINT data_source_condition_data_source_object_id_foreign
    REFERENCES data_source_object
    ON DELETE CASCADE,
  name                  VARCHAR(255)                    NOT NULL,
  type                  VARCHAR(255)                    NOT NULL,
  field                 VARCHAR(255),
  value                 VARCHAR(255),
  execute_on_query      BOOLEAN,
  execute_on_insert     BOOLEAN DEFAULT FALSE,
  execute_on_update     BOOLEAN DEFAULT FALSE,
  CONSTRAINT data_source_condition_name_data_source_object_id_unique
  UNIQUE (name, data_source_object_id)
);

-- auto-generated definition
CREATE TABLE data_source_condition_permission
(
  id                       UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_condition_permission_pkey
    PRIMARY KEY,
  organization_id          UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_permission_organization_id_foreign
    --REFERENCES organization,
  created_by_id            UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_permission_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id            UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_permission_updated_by_id_foreign
    --REFERENCES "user",
  created_at               TIMESTAMP WITH TIME ZONE,
  updated_at               TIMESTAMP WITH TIME ZONE,
  permission_set_id        UUID                            NOT NULL,
    --CONSTRAINT data_source_condition_permission_permission_set_id_foreign
    --REFERENCES permission_set
    --ON DELETE CASCADE,
  data_source_condition_id UUID                            NOT NULL
    CONSTRAINT data_source_condition_permission_data_source_condition_id_forei
    REFERENCES data_source_condition
    ON DELETE CASCADE,
  always_on                BOOLEAN,
  CONSTRAINT data_source_condition_permission_permission_set_id_data_source_
  UNIQUE (permission_set_id, data_source_condition_id)
);

-- auto-generated definition
CREATE TABLE data_source_field
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_field_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_field_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_field_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_field_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  data_source_object_id UUID                            NOT NULL
    CONSTRAINT data_source_field_data_source_object_id_foreign
    REFERENCES data_source_object
    ON DELETE CASCADE,
  name                  VARCHAR(255)                    NOT NULL,
  display_type          VARCHAR(255),
  label                 VARCHAR(255),
  length                INTEGER,
  filterable            BOOLEAN,
  groupable             BOOLEAN,
  required              BOOLEAN,
  readonly              BOOLEAN,
  sortable              BOOLEAN,
  reference_to          JSONB,
  is_id_field           BOOLEAN,
  is_name_field         BOOLEAN,
  CONSTRAINT data_source_field_name_data_source_object_id_unique
  UNIQUE (name, data_source_object_id)
);

-- auto-generated definition
CREATE TABLE data_source_field_permission
(
  id                   UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_field_permission_pkey
    PRIMARY KEY,
  organization_id      UUID                            NOT NULL,
    --CONSTRAINT data_source_field_permission_organization_id_foreign
    --REFERENCES organization,
  created_by_id        UUID                            NOT NULL,
    --CONSTRAINT data_source_field_permission_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id        UUID                            NOT NULL,
    --CONSTRAINT data_source_field_permission_updated_by_id_foreign
    --REFERENCES "user",
  created_at           TIMESTAMP WITH TIME ZONE,
  updated_at           TIMESTAMP WITH TIME ZONE,
  permission_set_id    UUID                            NOT NULL,
    --CONSTRAINT data_source_field_permission_permission_set_id_foreign
    --REFERENCES permission_set
    --ON DELETE CASCADE,
  data_source_field_id UUID                            NOT NULL
    CONSTRAINT data_source_field_permission_data_source_field_id_foreign
    REFERENCES data_source_field
    ON DELETE CASCADE,
  createable           BOOLEAN,
  deleteable           BOOLEAN,
  queryable            BOOLEAN,
  updateable           BOOLEAN,
  CONSTRAINT data_source_field_permission_permission_set_id_data_source_fiel
  UNIQUE (permission_set_id, data_source_field_id)
);

-- auto-generated definition
CREATE TABLE data_source_action
(
  id              UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_action_pkey
    PRIMARY KEY,
  organization_id UUID                            NOT NULL,
    --CONSTRAINT data_source_action_organization_id_foreign
    --REFERENCES organization,
  created_by_id   UUID                            NOT NULL,
    --CONSTRAINT data_source_action_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id   UUID                            NOT NULL,
    --CONSTRAINT data_source_action_updated_by_id_foreign
    --REFERENCES "user",
  created_at      TIMESTAMP WITH TIME ZONE,
  updated_at      TIMESTAMP WITH TIME ZONE,
  data_source_id  UUID                            NOT NULL
    CONSTRAINT data_source_action_data_source_id_foreign
    REFERENCES data_source
    ON DELETE CASCADE,
  name            VARCHAR(255)                    NOT NULL,
  label           VARCHAR(255),
  custom_config   JSONB
);

-- auto-generated definition
CREATE TABLE data_source_action_input
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_action_input_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_action_input_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_input_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_input_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  data_source_action_id UUID                            NOT NULL
    CONSTRAINT data_source_action_input_data_source_action_id_foreign
    REFERENCES data_source_action
    ON DELETE CASCADE,
  name                  VARCHAR(255)                    NOT NULL,
  display_type          VARCHAR(255)                    NOT NULL,
  label                 VARCHAR(255),
  index                 INTEGER
);

-- auto-generated definition
CREATE TABLE data_source_action_output
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_action_output_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_action_output_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_output_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_output_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  data_source_action_id UUID                            NOT NULL
    CONSTRAINT data_source_action_output_data_source_action_id_foreign
    REFERENCES data_source_action
    ON DELETE CASCADE,
  name                  VARCHAR(255)                    NOT NULL,
  display_type          VARCHAR(255),
  label                 VARCHAR(255),
  index                 INTEGER
);

-- auto-generated definition
CREATE TABLE data_source_action_permission
(
  id                    UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT data_source_action_permission_pkey
    PRIMARY KEY,
  organization_id       UUID                            NOT NULL,
    --CONSTRAINT data_source_action_permission_organization_id_foreign
    --REFERENCES organization,
  created_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_permission_created_by_id_foreign
    --REFERENCES "user",
  updated_by_id         UUID                            NOT NULL,
    --CONSTRAINT data_source_action_permission_updated_by_id_foreign
    --REFERENCES "user",
  created_at            TIMESTAMP WITH TIME ZONE,
  updated_at            TIMESTAMP WITH TIME ZONE,
  permission_set_id     UUID                            NOT NULL,
    --CONSTRAINT data_source_action_permission_permission_set_id_foreign
    --REFERENCES permission_set,
  data_source_action_id UUID                            NOT NULL
    CONSTRAINT data_source_action_permission_data_source_action_id_foreign
    REFERENCES data_source_action
    ON DELETE CASCADE,
  executable            BOOLEAN
);
