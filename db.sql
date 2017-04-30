-- For fast local dev, drop all tables we are working on.
-- TODO: Remove this in favor of clean migrations.
DROP TABLE IF EXISTS users                  CASCADE;
DROP TABLE IF EXISTS organizations          CASCADE;
DROP TABLE IF EXISTS teams                  CASCADE;
DROP TABLE IF EXISTS payment_plans          CASCADE;
DROP TABLE IF EXISTS user_organization_join CASCADE;
DROP TABLE IF EXISTS user_team_join         CASCADE;

-- NOTE: We explicitly declare the primary keys as "NOT NULL" in case we switch to mysql later.

CREATE TABLE users (
  user_id  UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),

  owner_id   UUID                     NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE organizations (
  organization_id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),

  owner_id   UUID                     NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE user_organization_join (
  organization_id UUID NOT NULL,
  user_id         UUID NOT NULL,

  user_role VARCHAR NOT NULL DEFAULT 'user',

  owner_id   UUID                     NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMP WITH TIME ZONE,

  PRIMARY KEY (organization_id, user_id)
);

-- Debug seed data.
INSERT INTO users (user_id, owner_id) VALUES (uuid_nil(), uuid_nil());
INSERT INTO organizations (organization_id, owner_id) VALUES (uuid_nil(), uuid_nil());

INSERT INTO user_organization_join (organization_id, user_id, user_role, owner_id)
VALUES (uuid_nil(), uuid_nil(), 'user', uuid_nil());
