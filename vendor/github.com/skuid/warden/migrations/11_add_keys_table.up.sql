CREATE TABLE site_jwt_key
(
  id              UUID DEFAULT uuid_generate_v4() NOT NULL
    CONSTRAINT site_jwt_key_pkey
    PRIMARY KEY,
  organization_id UUID                            NOT NULL,
  created_by_id   UUID                            NOT NULL,
  updated_by_id   UUID                            NOT NULL,
  created_at      TIMESTAMP WITH TIME ZONE,
  updated_at      TIMESTAMP WITH TIME ZONE,
  public_key      TEXT                            NOT NULL
);