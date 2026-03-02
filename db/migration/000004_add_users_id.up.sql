ALTER TABLE users
ADD COLUMN id BIGINT;

CREATE SEQUENCE IF NOT EXISTS users_id_seq;

ALTER TABLE users
ALTER COLUMN id SET DEFAULT nextval('users_id_seq');

ALTER TABLE users
ADD CONSTRAINT users_id_unique UNIQUE (id);
